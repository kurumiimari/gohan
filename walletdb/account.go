package walletdb

import (
	"github.com/kurumiimari/gohan/chain"
	"github.com/kurumiimari/gohan/shakedex"
	"github.com/pkg/errors"
)

type AccountOpts struct {
	ID              string
	Seed            string
	WatchOnly       bool
	Idx             uint32
	ChangeIdx       uint32
	RecvIdx         uint32
	DutchAuctionIdx uint32
	XPub            chain.ExtendedKey
	RescanHeight    int
	AddressBloom    []byte
	OutpointBloom   []byte
	LookaheadTips   map[uint32]uint32
}

func CreateAccount(
	tx Transactor,
	opts *AccountOpts,
) error {
	_, err := tx.Exec(`
INSERT INTO accounts (
	id, 
	seed, 
	watch_only, 
	idx,
	recv_idx,
	change_idx,
	dutch_auction_idx,
	xpub,
	rescan_height,
	address_bloom, 
	outpoint_bloom
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`,
		opts.ID,
		opts.Seed,
		opts.WatchOnly,
		opts.Idx,
		opts.RecvIdx,
		opts.ChangeIdx,
		opts.DutchAuctionIdx,
		opts.XPub.PublicString(),
		opts.RescanHeight,
		opts.AddressBloom,
		opts.OutpointBloom,
	)
	return errors.WithStack(err)
}

func GetAllAccounts(q Querier) ([]*AccountOpts, error) {
	rows, err := q.Query(`
SELECT
	id, 
	seed, 
	watch_only, 
	idx,
	recv_idx,
	change_idx,
	dutch_auction_idx,
	xpub,
	rescan_height,
	address_bloom, 
	outpoint_bloom
FROM accounts ORDER BY id
`,
	)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer rows.Close()
	var out []*AccountOpts

	for rows.Next() {
		opts, err := scanAccountOpts(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, opts)
	}

	for _, acc := range out {
		tips, err := GetLookaheadTips(q, acc.ID)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		acc.LookaheadTips = tips
	}

	return out, errors.WithStack(rows.Err())
}

func GetAccount(q Querier, id string) (*AccountOpts, error) {
	row := q.QueryRow(`
SELECT
	id, 
	seed, 
	watch_only, 
	idx,
	recv_idx,
	change_idx,
	dutch_auction_idx,
	xpub,
	rescan_height,
	address_bloom, 
	outpoint_bloom
FROM accounts
WHERE id = ?
`,
		id,
	)
	if row.Err() != nil {
		return nil, errors.WithStack(row.Err())
	}
	return scanAccountOpts(row)
}

func UpdateAddressIdx(tx Transactor, accountID string, branch, idx uint32) error {
	var query string
	switch branch {
	case chain.ReceiveBranch:
		query = "UPDATE accounts SET recv_idx = ? WHERE id = ?"
	case chain.ChangeBranch:
		query = "UPDATE accounts SET change_idx = ? WHERE id = ?"
	case shakedex.AddressBranch:
		query = "UPDATE accounts SET dutch_auction_idx = ? WHERE id = ?"
	default:
		return errors.New("unsupported address branch")
	}

	_, err := tx.Exec(query, idx, accountID)
	return errors.WithStack(err)
}

func UpdateAddressBloom(tx Transactor, accountID string, bloom []byte) error {
	_, err := tx.Exec(
		"UPDATE accounts SET address_bloom = ? WHERE id = ?",
		bloom,
		accountID,
	)
	return errors.WithStack(err)
}

func UpdateOutpointBloom(tx Transactor, accountID string, bloom []byte) error {
	_, err := tx.Exec(
		"UPDATE accounts SET outpoint_bloom = ? WHERE id = ?",
		bloom,
		accountID,
	)
	return errors.WithStack(err)
}

func UpdateRescanHeight(tx Transactor, accountID string, rescanHeight int) error {
	_, err := tx.Exec(
		"UPDATE accounts SET rescan_height = ? WHERE id = ?",
		rescanHeight,
		accountID,
	)
	return errors.WithStack(err)
}

type Balances struct {
	Available    uint64 `json:"available"`
	Immature     uint64 `json:"immature"`
	BidLocked    uint64 `json:"bid_locked"`
	RevealLocked uint64 `json:"reveal_locked"`
	NameLocked   uint64 `json:"name_locked"`
}

func GetBalances(tx Transactor, accountID string, network *chain.Network, height int) (*Balances, error) {
	available, err := scanBalance(tx, `
SELECT COALESCE(SUM(value), 0) FROM coins
JOIN transactions ON (transactions.hash = coins.tx_hash AND transactions.account_id = coins.account_id)
WHERE spending_tx_hash IS NULL
AND (covenant_type = ? OR covenant_type = ?)
AND coins.account_id = ?
AND (coinbase = FALSE OR (coinbase = TRUE AND transactions.block_height <= ?))
`,
		uint8(chain.CovenantNone),
		uint8(chain.CovenantRedeem),
		accountID,
		height-network.CoinbaseMaturity,
	)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	immature, err := scanBalance(tx, `
SELECT COALESCE(SUM(value), 0) FROM coins
JOIN transactions ON (transactions.hash = coins.tx_hash AND transactions.account_id = coins.account_id)
WHERE spending_tx_hash IS NULL
AND covenant_type = ? 
AND coins.account_id = ?
AND coinbase = TRUE 
AND transactions.block_height > ?
`,
		uint8(chain.CovenantNone),
		accountID,
		height-network.CoinbaseMaturity,
	)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	bidLocked, err := scanBalance(tx, `
SELECT COALESCE(SUM(value), 0) FROM coins
JOIN transactions ON (transactions.hash = coins.tx_hash AND transactions.account_id = coins.account_id)
WHERE spending_tx_hash IS NULL
AND covenant_type = ?
AND coins.account_id = ?
`,
		uint8(chain.CovenantBid),
		accountID,
	)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	revealLocked, err := scanBalance(tx, `
SELECT COALESCE(SUM(value), 0) FROM coins
JOIN transactions ON (transactions.hash = coins.tx_hash AND transactions.account_id = coins.account_id)
WHERE spending_tx_hash IS NULL
AND covenant_type = ? 
AND coins.account_id = ?
`,
		uint8(chain.CovenantReveal),
		accountID,
	)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	nameLocked, err := scanBalance(tx, `
SELECT COALESCE(SUM(value), 0) FROM coins
JOIN transactions ON (transactions.hash = coins.tx_hash AND transactions.account_id = coins.account_id)
WHERE spending_tx_hash IS NULL
AND (
	covenant_type = ? OR 
	covenant_type = ? OR 
	covenant_type = ? OR
	covenant_type = ? OR
	covenant_type = ?
) 
AND coins.account_id = ?
`,
		uint8(chain.CovenantUpdate),
		uint8(chain.CovenantRegister),
		uint8(chain.CovenantTransfer),
		uint8(chain.CovenantFinalize),
		uint8(chain.CovenantRevoke),
		accountID,
	)

	return &Balances{
		Available:    available,
		Immature:     immature,
		BidLocked:    bidLocked,
		RevealLocked: revealLocked,
		NameLocked:   nameLocked,
	}, nil
}

func Rollback(tx Transactor, accountID string, height int) error {
	_, err := tx.Exec(`
DELETE FROM name_history
WHERE ROWID IN (
	SELECT name_history.ROWID FROM name_history
	JOIN transactions ON (transactions.hash = name_history.tx_hash AND transactions.account_id = name_history.account_id)
	WHERE transactions.block_height > ? AND name_history.account_id = ?	
)
`,
		height,
		accountID,
	)
	if err != nil {
		return errors.WithStack(err)
	}

	_, err = tx.Exec(`
DELETE FROM coins
WHERE ROWID IN (
	SELECT coins.ROWID FROM coins
	JOIN transactions ON (transactions.hash = coins.tx_hash AND transactions.account_id = coins.account_id)
	WHERE transactions.block_height > ? AND coins.account_id = ?	
)
`,
		height,
		accountID,
	)
	if err != nil {
		return errors.WithStack(err)
	}

	_, err = tx.Exec(`
UPDATE transactions SET 
block_height = -1, block_hash = '0000000000000000000000000000000000000000000000000000000000000000'
WHERE account_id = ? AND block_height > ?
`,
		accountID,
		height,
	)
	if err != nil {
		return errors.WithStack(err)
	}

	_, err = tx.Exec("UPDATE accounts SET rescan_height = ? WHERE id = ?", height, accountID)
	return errors.WithStack(err)
}

func scanBalance(tx Transactor, query string, queryArgs ...interface{}) (uint64, error) {
	row := tx.QueryRow(
		query,
		queryArgs...,
	)
	if row.Err() != nil {
		return 0, errors.WithStack(row.Err())
	}
	var bal uint64
	if err := row.Scan(&bal); err != nil {
		return 0, errors.WithStack(err)
	}
	return bal, nil
}

func scanAccountOpts(scanner Scanner) (*AccountOpts, error) {
	var err error
	var xPubStr string

	opts := new(AccountOpts)
	err = scanner.Scan(
		&opts.ID,
		&opts.Seed,
		&opts.WatchOnly,
		&opts.Idx,
		&opts.ChangeIdx,
		&opts.RecvIdx,
		&opts.DutchAuctionIdx,
		&xPubStr,
		&opts.RescanHeight,
		&opts.AddressBloom,
		&opts.OutpointBloom,
	)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	opts.XPub, err = chain.NewMasterExtendedKeyFromXPub(xPubStr, chain.GetCurrNetwork())
	if err != nil {
		return nil, err
	}
	return opts, nil
}
