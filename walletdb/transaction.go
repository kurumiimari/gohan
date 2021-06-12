package walletdb

import (
	"bytes"
	"database/sql"
	"encoding/hex"
	"github.com/kurumiimari/gohan/chain"
	"github.com/kurumiimari/gohan/gcrypto"
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

func GetTransactionByOutpoint(q Transactor, accountID string, hash gcrypto.Hash) (*Transaction, error) {
	row := q.QueryRow(`
SELECT
	hash,
	idx,
	block_height,
	block_hash,
	raw,
	time
FROM transactions WHERE account_id = ? AND hash = ?
`,
		accountID,
		hash.String(),
	)
	if row.Err() != nil {
		return nil, errors.WithStack(row.Err())
	}

	tx := new(Transaction)
	err := row.Scan(
		&tx.Hash,
		&tx.Idx,
		&tx.BlockHeight,
		&tx.BlockHash,
		&tx.Raw,
		&tx.Time,
	)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return tx, nil
}

type RichTransaction struct {
	Hash    gcrypto.Hash  `json:"hash"`
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
	Prevout  *chain.Outpoint `json:"prevout"`
	Witness  *chain.Witness  `json:"witness"`
	Sequence uint32          `json:"sequence"`
	Coin     *RichCoin       `json:"coin"`
}

type RichOutput struct {
	Value    uint64          `json:"value"`
	Address  *RichAddress    `json:"address"`
	Covenant *chain.Covenant `json:"covenant"`
}

type RichCoin struct {
	Version  int             `json:"version"`
	Height   int             `json:"height"`
	Value    uint64          `json:"value"`
	Address  *RichAddress    `json:"address"`
	Covenant *chain.Covenant `json:"covenant"`
	Coinbase bool            `json:"coinbase"`
}

type RichAddress struct {
	Address    *chain.Address   `json:"address"`
	Derivation chain.Derivation `json:"derivation"`
	Own        bool             `json:"own"`
}

func ListTransactions(q Querier, accountID string, count, offset int) ([]*RichTransaction, error) {
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
		out, err = inflateTx(q, accountID, txRows, out)
		if err != nil {
			return nil, errors.WithStack(err)
		}
	}
	return out, nil
}

func inflateTx(q Querier, accountID string, r *sql.Rows, out []*RichTransaction) ([]*RichTransaction, error) {
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
	if err := fillRichTransaction(q, accountID, tx, raw); err != nil {
		return nil, err
	}
	return append(out, tx), nil
}

func fillRichTransaction(q Querier, accountID string, tx *RichTransaction, rawTx []byte) error {
	protoTx := new(chain.Transaction)
	if _, err := protoTx.ReadFrom(bytes.NewReader(rawTx)); err != nil {
		panic(err)
	}

	var totalInputs uint64
	var totalOutputs uint64
	hasAllInputs := true
	for i, input := range protoTx.Inputs {
		coin, err := GetCoinByPrevout(q, accountID, input.Prevout)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		var rc *RichCoin
		if coin == nil {
			hasAllInputs = false
		} else {
			rc = &RichCoin{
				Version: 0,
				Height:  coin.Height,
				Value:   coin.Value,
				Address: &RichAddress{
					Address:    coin.Address,
					Derivation: coin.Derivation,
					Own:        true,
				},
				Covenant: coin.Covenant,
				Coinbase: coin.Coinbase,
			}
			totalInputs += coin.Value
		}

		tx.Inputs = append(tx.Inputs, &RichInput{
			Prevout:  input.Prevout,
			Witness:  protoTx.Witnesses[i],
			Sequence: input.Sequence,
			Coin:     rc,
		})
	}

	for i, output := range protoTx.Outputs {
		coin, err := GetCoinByPrevout(q, accountID, &chain.Outpoint{
			Hash:  tx.Hash,
			Index: uint32(i),
		})
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		var deriv chain.Derivation
		if coin != nil {
			deriv = coin.Derivation
		}

		tx.Outputs = append(tx.Outputs, &RichOutput{
			Value: output.Value,
			Address: &RichAddress{
				Address:    output.Address,
				Derivation: deriv,
				Own:        coin != nil,
			},
			Covenant: output.Covenant,
		})

		totalOutputs += output.Value
	}

	if hasAllInputs {
		tx.Fee = totalInputs - totalOutputs
	}

	return nil
}
