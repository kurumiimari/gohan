package walletdb

import (
	"bytes"
	"database/sql/driver"
	"github.com/kurumiimari/gohan/bio"
	"github.com/kurumiimari/gohan/chain"
	"github.com/kurumiimari/gohan/gcrypto"
	"github.com/pkg/errors"
)

type CoinType uint8

const (
	CoinTypeDefault CoinType = iota
	CoinTypeDutchAuctionListing
	CoinTypeDutchAuctionFill
	CoinTypeDutchAuctionCancel
)

type Coin struct {
	AccountID string
	Spent     bool
	NameHash  gcrypto.Hash
	Type      CoinType

	Height     int
	Value      uint64
	Address    *chain.Address
	Covenant   *chain.Covenant
	Prevout    *chain.Outpoint
	Coinbase   bool
	Derivation chain.Derivation
}

func (c *Coin) AsChain() *chain.Coin {
	return &chain.Coin{
		Version:    0,
		Height:     c.Height,
		Value:      c.Value,
		Address:    c.Address,
		Covenant:   c.Covenant,
		Prevout:    c.Prevout,
		Coinbase:   c.Coinbase,
		Derivation: c.Derivation,
	}
}

type SQLCovenantItems [][]byte

func (s *SQLCovenantItems) Scan(src interface{}) error {
	switch src.(type) {
	case nil:
		*s = make([][]byte, 0)
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

func CreateCoin(
	tx Transactor,
	accountID string,
	prevout *chain.Outpoint,
	value uint64,
	address *chain.Address,
	covenant *chain.Covenant,
	coinbase bool,
	coinType CoinType,
) error {
	var nameHash gcrypto.Hash
	if covenant.Type > 1 {
		nameHash = covenant.Items[0]
	}

	_, err := tx.Exec(`
INSERT INTO coins(
	account_id,
	tx_hash,
	out_idx,
	value,
	address,
	covenant_type,
	covenant_items,
	coinbase,
	name_hash,
	type
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT (account_id, tx_hash, out_idx) DO NOTHING
`,
		accountID,
		prevout.Hash.String(),
		prevout.Index,
		value,
		address,
		uint8(covenant.Type),
		SQLCovenantItems(covenant.Items),
		coinbase,
		nameHash,
		coinType,
	)
	return errors.WithStack(err)
}

func GetCoinByPrevout(tx Transactor, accountID string, prevout *chain.Outpoint) (*Coin, error) {
	row := tx.QueryRow(
		coinQuery("WHERE coins.account_id = ? AND tx_hash = ? AND out_idx = ?"),
		accountID,
		prevout.Hash.String(),
		prevout.Index,
	)
	if row.Err() != nil {
		return nil, errors.WithStack(row.Err())
	}
	return scanCoin(row)
}

func GetDutchAuctionCoin(tx Transactor, accountID string, prevout *chain.Outpoint) (*Coin, error) {
	row := tx.QueryRow(`
SELECT 
	coins.account_id AS account_id,
	txout.block_hash IS NOT NULL as spent,
	coins.name_hash AS name_hash,
	coins.type AS type,
	txin.block_height AS height,
	coins.value as VALUE,
	coins.address AS address,
	coins.covenant_type AS covenant_type,
	coins.covenant_items AS covenant_items,
	coins.tx_hash AS tx_hash, 
	coins.out_idx AS out_idx,
	coins.coinbase AS coinbase,
	0 as branch,
	0 as idx
FROM coins
INNER JOIN transactions AS txin ON txin.account_id = coins.account_id AND txin.hash = coins.tx_hash
LEFT JOIN transactions AS txout ON txout.account_id = coins.account_id AND txout.hash = coins.spending_tx_hash
WHERE coins.account_id = ?
AND tx_hash = ?
AND out_idx = ?
`,
		accountID,
		prevout.Hash.String(),
		prevout.Index,
	)
	if row.Err() != nil {
		return nil, errors.WithStack(row.Err())
	}
	coin, err := scanCoin(row)
	if err != nil {
		return nil, err
	}
	coin.Derivation = nil
	return coin, nil
}

func UpdateCoinSpent(tx Transactor, outpoint *chain.Outpoint, spendHash gcrypto.Hash) error {
	_, err := tx.Exec(
		"UPDATE coins SET spending_tx_hash = ? WHERE tx_hash = ? AND out_idx = ?",
		spendHash,
		outpoint.Hash.String(),
		outpoint.Index,
	)
	return errors.WithStack(err)
}

func GetFundingCoins(tx Transactor, accountID string, network *chain.Network, height int) ([]*Coin, error) {
	rows, err := tx.Query(
		coinQuery(`
WHERE coins.spending_tx_hash IS NULL
AND (
	(txin.block_height <= ? AND coins.coinbase = TRUE) OR
	(coins.coinbase = FALSE)
)
AND coins.covenant_type = ? 
AND coins.account_id = ?
AND coins.type = ?
ORDER BY value ASC
`),
		height-network.CoinbaseMaturity,
		uint8(chain.CovenantNone),
		accountID,
		uint8(CoinTypeDefault),
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

func GetFinalizableDutchAuctionFillCoin(q Querier, accountID, name string) (*Coin, *Address, error) {
	row := q.QueryRow(
		`
SELECT 
	coins.account_id AS account_id,
	txout.block_hash IS NOT NULL as spent,
	coins.name_hash AS name_hash,
	coins.type AS type,
	txin.block_height AS height,
	coins.value as VALUE,
	coins.address AS address,
	coins.covenant_type AS covenant_type,
	coins.covenant_items AS covenant_items,
	coins.tx_hash AS tx_hash, 
	coins.out_idx AS out_idx,
	coins.coinbase AS coinbase,
	0 as branch,
	0 as idx
FROM coins
INNER JOIN transactions AS txin ON txin.account_id = coins.account_id AND txin.hash = coins.tx_hash
LEFT JOIN transactions AS txout ON txout.account_id = coins.account_id AND txout.hash = coins.spending_tx_hash
WHERE coins.spending_tx_hash IS NULL  
AND coins.account_id = ?
AND coins.type = ?
AND coins.covenant_type = ?
AND coins.name_hash = ?
`,
		accountID,
		uint8(CoinTypeDutchAuctionFill),
		uint8(chain.CovenantTransfer),
		chain.HashName(name),
	)
	if row.Err() != nil {
		return nil, nil, errors.WithStack(row.Err())
	}
	coin, err := scanCoin(row)
	if err != nil {
		return nil, nil, err
	}
	addr, err := GetAddress(q, accountID, &chain.Address{
		Version: coin.Covenant.Items[2][0],
		Hash:    coin.Covenant.Items[3],
	})
	if err != nil {
		return nil, nil, err
	}
	coin.Derivation = nil
	return coin, addr, nil
}

func GetTransferrableDutchAuctionCancelCoin(q Querier, accountID, name string) (*Coin, error) {
	row := q.QueryRow(
		coinQuery(`
WHERE coins.spending_tx_hash IS NULL
AND coins.account_id = ?
AND coins.type = ?
AND coins.covenant_type = ?
AND coins.name_hash = ?
`),
		accountID,
		uint8(CoinTypeDutchAuctionListing),
		uint8(chain.CovenantFinalize),
		chain.HashName(name),
	)
	if row.Err() != nil {
		return nil, errors.WithStack(row.Err())
	}
	return scanCoin(row)
}

func GetFinalizableDutchAuctionCancelCoin(q Querier, accountID, name string) (*Coin, error) {
	row := q.QueryRow(
		coinQuery(`
WHERE coins.spending_tx_hash IS NULL
AND coins.account_id = ?
AND coins.type = ?
AND coins.covenant_type = ?
AND coins.name_hash = ?
`),
		accountID,
		uint8(CoinTypeDutchAuctionCancel),
		uint8(chain.CovenantTransfer),
		chain.HashName(name),
	)
	if row.Err() != nil {
		return nil, errors.WithStack(row.Err())
	}
	return scanCoin(row)
}

func scanCoin(s Scanner, addlFields ...interface{}) (*Coin, error) {
	coin := new(Coin)
	coin.Prevout = new(chain.Outpoint)
	var covType uint8
	var covItems SQLCovenantItems
	var branch uint32
	var index uint32
	err := s.Scan(append([]interface{}{
		&coin.AccountID,
		&coin.Spent,
		&coin.NameHash,
		&coin.Type,
		&coin.Height,
		&coin.Value,
		&coin.Address,
		&covType,
		&covItems,
		&coin.Prevout.Hash,
		&coin.Prevout.Index,
		&coin.Coinbase,
		&branch,
		&index,
	}, addlFields...)...)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	coin.Covenant = &chain.Covenant{
		Type:  chain.CovenantType(covType),
		Items: covItems,
	}
	coin.Derivation = chain.Derivation{branch, index}
	return coin, nil
}

const baseCoinQuery = `
SELECT 
	coins.account_id AS account_id,
	txout.block_hash IS NOT NULL as spent,
	coins.name_hash AS name_hash,
	coins.type AS type,
	txin.block_height AS height,
	coins.value as VALUE,
	coins.address AS address,
	coins.covenant_type AS covenant_type,
	coins.covenant_items AS covenant_items,
	coins.tx_hash AS tx_hash, 
	coins.out_idx AS out_idx,
	coins.coinbase AS coinbase,
	addr.branch AS address_branch,
	addr.idx AS address_index
FROM coins
INNER JOIN addresses AS addr ON addr.account_id = coins.account_id AND addr.address = coins.address
INNER JOIN transactions AS txin ON txin.account_id = coins.account_id AND txin.hash = coins.tx_hash
LEFT JOIN transactions AS txout ON txout.account_id = coins.account_id AND txout.hash = coins.spending_tx_hash
`

func coinQuery(fragment string) string {
	return baseCoinQuery + " " + fragment
}
