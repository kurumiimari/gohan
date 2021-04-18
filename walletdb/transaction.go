package walletdb

import (
	"bytes"
	"database/sql"
	"encoding/hex"
	"github.com/kurumiimari/gohan/chain"
	"github.com/pkg/errors"
)

type Transaction struct {
	Hash        string
	Idx         int
	BlockHeight int
	BlockHash   string
	Raw         []byte
	Time        int
}

func UpsertTransaction(tx Transactor, accountID string, txObj *Transaction) (*Transaction, error) {
	_, err := tx.Exec(`
INSERT INTO transactions (
	account_id,
	hash,
	idx,
	block_height,
	block_hash,
	raw,
	time
) VALUES(?, ?, ?, ?, ?, ?, ?) ON CONFLICT (account_id, hash) DO UPDATE SET block_height = ?, idx = ?, block_hash = ?, time = ?
`,
		accountID,
		txObj.Hash,
		txObj.Idx,
		txObj.BlockHeight,
		txObj.BlockHash,
		txObj.Raw,
		txObj.Time,
		txObj.BlockHeight,
		txObj.Idx,
		txObj.BlockHash,
		txObj.Time,
	)
	return txObj, errors.WithStack(err)
}

type RichTransaction struct {
	Hash    string        `json:"hash"`
	Height  int           `json:"height"`
	Block   string        `json:"block"`
	Time    int           `json:"time"`
	Index   int           `json:"index"`
	Version int           `json:"version"`
	Inputs  []*RichInput  `json:"inputs"`
	Outputs []*RichOutput `json:"outputs"`
	Hex     string        `json:"hex"`
	Fee     uint64        `json:"fee"`
}

type RichInput struct {
	Prevout  *RichPrevout `json:"prevout"`
	Witness  []string     `json:"witness"`
	Sequence uint32       `json:"sequence"`
	Coin     *RichCoin    `json:"coin"`
}

type RichOutput struct {
	Value    uint64        `json:"value"`
	Address  *RichAddress  `json:"address"`
	Covenant *RichCovenant `json:"covenant"`
}

type RichPrevout struct {
	Hash  string `json:"hash"`
	Index uint32 `json:"index"`
}

type RichCoin struct {
	Version  int           `json:"version"`
	Height   int           `json:"height"`
	Value    uint64        `json:"value"`
	Address  *RichAddress  `json:"address"`
	Covenant *RichCovenant `json:"covenant"`
	Coinbase bool          `json:"coinbase"`
}

type RichAddress struct {
	Address    string           `json:"address"`
	Derivation chain.Derivation `json:"derivation"`
	Own        bool             `json:"own"`
}

type RichCovenant struct {
	Type   int      `json:"type"`
	Action string   `json:"action"`
	Items  []string `json:"items"`
}

func ListTransactions(q Querier, accountID string, network *chain.Network, count, offset int) ([]*RichTransaction, error) {
	txRows, err := q.Query(`
SELECT
	hash,
	idx,
	block_height,
	block_hash,
	raw,
	time
FROM transactions
WHERE account_id = ?
ORDER BY block_height DESC LIMIT ? OFFSET ?
`,
		accountID,
		count,
		offset,
	)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer txRows.Close()

	out := make([]*RichTransaction, 0)
	for txRows.Next() {
		out, err = inflateTx(q, accountID, network, txRows, out)
		if err != nil {
			return nil, errors.WithStack(err)
		}
	}
	return out, nil
}

func inflateTx(q Querier, accountID string, network *chain.Network, r *sql.Rows, out []*RichTransaction) ([]*RichTransaction, error) {
	tx := new(RichTransaction)
	var raw []byte
	err := r.Scan(
		&tx.Hash,
		&tx.Index,
		&tx.Height,
		&tx.Block,
		&raw,
		&tx.Time,
	)
	if err != nil {
		return nil, err
	}
	tx.Hex = hex.EncodeToString(raw)
	if err := fillRichTransaction(q, accountID, network, tx, raw); err != nil {
		return nil, err
	}
	return append(out, tx), nil
}

func fillRichTransaction(q Querier, accountID string, network *chain.Network, tx *RichTransaction, rawTx []byte) error {
	protoTx := new(chain.Transaction)
	if _, err := protoTx.ReadFrom(bytes.NewReader(rawTx)); err != nil {
		panic(err)
	}

	var totalInputs uint64
	var totalOutputs uint64
	hasAllInputs := true
	for i, input := range protoTx.Inputs {
		witnesses := convertItemsToHex(protoTx.Witnesses[i].Items)
		pHashStr := hex.EncodeToString(input.Prevout.Hash)
		coin, err := GetCoinByOutpoint(q, accountID, pHashStr, int(input.Prevout.Index))
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		var rc *RichCoin
		if coin == nil {
			hasAllInputs = false
		} else {
			rc = &RichCoin{
				Version: 0,
				Height:  coin.BlockHeight,
				Value:   coin.Value,
				Address: &RichAddress{
					Address:    coin.Address,
					Derivation: chain.Derivation{coin.AddressBranch, coin.AddressIndex},
					Own:        true,
				},
				Covenant: &RichCovenant{
					Type:   int(chain.NewCovenantTypeFromString(coin.CovenantType)),
					Action: coin.CovenantType,
					Items:  convertItemsToHex(coin.CovenantItems),
				},
				Coinbase: coin.Coinbase,
			}
			totalInputs += coin.Value
		}

		tx.Inputs = append(tx.Inputs, &RichInput{
			Prevout: &RichPrevout{
				Hash:  pHashStr,
				Index: input.Prevout.Index,
			},
			Witness:  witnesses,
			Sequence: input.Sequence,
			Coin:     rc,
		})
	}

	for i, output := range protoTx.Outputs {
		coin, err := GetCoinByOutpoint(q, accountID, tx.Hash, i)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}

		var deriv chain.Derivation
		if coin != nil {
			deriv = chain.Derivation{coin.AddressBranch, coin.AddressIndex}
		}

		tx.Outputs = append(tx.Outputs, &RichOutput{
			Value: output.Value,
			Address: &RichAddress{
				Address:    output.Address.String(network),
				Derivation: deriv,
				Own:        coin != nil,
			},
			Covenant: &RichCovenant{
				Type:   int(output.Covenant.Type),
				Action: output.Covenant.Type.String(),
				Items:  convertItemsToHex(output.Covenant.Items),
			},
		})

		totalOutputs += output.Value
	}

	if hasAllInputs {
		tx.Fee = totalInputs - totalOutputs
	}

	return nil
}

func convertItemsToHex(in [][]byte) []string {
	out := make([]string, len(in))
	for i, item := range in {
		out[i] = hex.EncodeToString(item)
	}
	return out
}
