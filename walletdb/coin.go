package walletdb

import (
	"bytes"
	"database/sql"
	"database/sql/driver"
	"github.com/kurumiimari/gohan/bio"
	"github.com/kurumiimari/gohan/chain"
	"github.com/pkg/errors"
)

type CreateCoinOpts struct {
	AccountID     string
	TxHash        string
	OutIdx        int
	Value         uint64
	Address       string
	Coinbase      bool
	CovenantType  string
	CovenantItems [][]byte
}

type Coin struct {
	ID                  int
	AccountID           string
	BlockHeight         int
	BlockHash           string
	TxIdx               int
	TxHash              string
	OutIdx              int
	Value               uint64
	Address             string
	AddressBranch       uint32
	AddressIndex        uint32
	Coinbase            bool
	CovenantType        string
	CovenantItems       [][]byte
	SpendingBlockHeight int
	SpendingBlockHash   string
	SpendingTxIdx       int
	SpendingTxHash      string
}

type SQLCovenantItems [][]byte

func (s *SQLCovenantItems) Scan(src interface{}) error {
	switch src.(type) {
	case nil:
		*s = nil
		return nil
	case []byte:
		b := src.([]byte)
		count := int(b[0])
		items := make([][]byte, count)
		g := bio.NewGuardReader(bytes.NewReader(b[1:]))
		for i := 0; i < count; i++ {
			items[i], _ = bio.ReadVarBytes(g)
		}
		if g.Err != nil {
			return g.Err
		}
		*s = items
		return nil
	default:
		return errors.New("incompatible type")
	}
}

func (s SQLCovenantItems) Value() (driver.Value, error) {
	if len(s) == 0 {
		return nil, nil
	}

	buf := new(bytes.Buffer)
	buf.WriteByte(byte(len(s)))
	for _, item := range s {
		bio.WriteVarBytes(buf, item)
	}
	return buf.Bytes(), nil
}

func CreateCoin(tx Transactor, opts *CreateCoinOpts) error {
	_, err := tx.Exec(`
INSERT INTO coins(
	account_id,
	tx_hash,
	out_idx,
	value,
	address,
	coinbase,
	covenant_type,
	covenant_items
) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
`,
		opts.AccountID,
		opts.TxHash,
		opts.OutIdx,
		opts.Value,
		opts.Address,
		opts.Coinbase,
		opts.CovenantType,
		SQLCovenantItems(opts.CovenantItems),
	)
	return errors.WithStack(err)
}

func GetCoinByOutpoint(tx Transactor, accountID, txHash string, outIdx int) (*Coin, error) {
	row := tx.QueryRow(
		coinQuery("WHERE coins.account_id = ? AND tx_hash = ? AND out_idx = ?"),
		accountID,
		txHash,
		outIdx,
	)
	if row.Err() != nil {
		return nil, errors.WithStack(row.Err())
	}
	return scanCoin(row)
}

func UpdateCoinSpent(tx Transactor, id int, txHash string) error {
	_, err := tx.Exec(
		"UPDATE coins SET spending_tx_hash = ? WHERE id = ?",
		txHash,
		id,
	)
	return errors.WithStack(err)
}

func GetFundingCoins(tx Transactor, accountID string, network *chain.Network, height int) ([]*Coin, error) {
	rows, err := tx.Query(
		coinQuery(`
WHERE spending_block_height IS NULL 
AND (
	(txin.block_height <= ? AND coins.coinbase = TRUE) OR
	(coins.coinbase = FALSE)
)
AND covenant_type = 'NONE' 
AND coins.account_id = ? 
ORDER BY value ASC
`),
		height-network.CoinbaseMaturity,
		accountID,
	)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer rows.Close()
	var coins []*Coin
	for rows.Next() {
		coin, err := scanCoin(rows)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		coins = append(coins, coin)
	}
	return coins, errors.WithStack(err)
}

func GetUnspentCoins(q Querier, accountID string) ([]*Coin, error) {
	rows, err := q.Query(
		coinQuery("WHERE spending_block_height IS NULL AND account_id = ? ORDER BY block_height, tx_idx, out_idx ASC"),
		accountID,
	)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer rows.Close()
	var coins []*Coin
	for rows.Next() {
		coin, err := scanCoin(rows)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		coins = append(coins, coin)
	}
	return coins, errors.WithStack(err)
}

func scanCoin(s Scanner) (*Coin, error) {
	coin := new(Coin)
	var covItems SQLCovenantItems
	var spendingBlockHeight sql.NullInt32
	var spendingBlockHash sql.NullString
	var spendingTxIdx sql.NullInt32
	var spendingTxHash sql.NullString
	err := s.Scan(
		&coin.ID,
		&coin.AccountID,
		&coin.BlockHeight,
		&coin.BlockHash,
		&coin.TxIdx,
		&coin.TxHash,
		&coin.OutIdx,
		&coin.Value,
		&coin.Address,
		&coin.AddressBranch,
		&coin.AddressIndex,
		&coin.Coinbase,
		&coin.CovenantType,
		&covItems,
		&spendingBlockHeight,
		&spendingBlockHash,
		&spendingTxIdx,
		&spendingTxHash,
	)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	coin.CovenantItems = covItems
	coin.SpendingBlockHeight = int(spendingBlockHeight.Int32)
	coin.SpendingBlockHash = spendingBlockHash.String
	coin.SpendingTxIdx = int(spendingTxIdx.Int32)
	coin.SpendingTxHash = spendingTxHash.String
	return coin, nil
}

const baseCoinQuery = `
SELECT 
	coins.id AS id,
	coins.account_id AS account_id,
	txin.block_height AS block_height,
	txin.block_hash AS block_hash, 
	txin.idx AS tx_idx, 
	coins.tx_hash AS tx_hash, 
	coins.out_idx AS out_idx, 
	coins.value AS value,
	coins.address AS address,
	addr.branch AS address_branch,
	addr.idx AS address_index, 
	coins.coinbase AS coinbase,
	coins.covenant_type AS covenant_type,
	coins.covenant_items AS covenant_items,
	txout.block_height AS spending_block_height,
	txout.block_hash AS spending_block_hash,
	txout.idx AS spending_tx_idx,
	coins.spending_tx_hash AS spending_tx_hash
FROM coins
INNER JOIN addresses AS addr ON addr.address = coins.address
INNER JOIN transactions AS txin ON txin.account_id = coins.account_id AND txin.hash = coins.tx_hash
LEFT JOIN transactions AS txout ON txout.account_id = coins.account_id AND txout.hash = coins.spending_tx_hash
`

func coinQuery(fragment string) string {
	return baseCoinQuery + " " + fragment
}
