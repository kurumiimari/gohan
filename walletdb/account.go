package walletdb

import (
	"fmt"
	"github.com/kurumiimari/gohan/chain"
	"github.com/pkg/errors"
)

type Account struct {
	ID            string
	Name          string
	WalletID      string
	Idx           uint32
	ChangeIdx     uint32
	ReceivingIdx  uint32
	XPub          string
	RescanHeight  int
	AddressBloom  []byte
	OutpointBloom []byte
	LookaheadTips [2]uint32
}

func CreateAccount(tx Transactor, name string, walletID string, xPub string, addrBloom []byte, outpointBloom []byte) (*Account, error) {
	account := new(Account)
	id := fmt.Sprintf("%s/%s", walletID, name)
	idx, err := GetNewAccountIdx(tx, walletID)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	_, err = tx.Exec(`
INSERT INTO accounts (id, name, wallet_id, idx, xpub, address_bloom, outpoint_bloom)
VALUES (?, ?, ?, ?, ?, ?, ?)
`,
		id,
		name,
		walletID,
		idx,
		xPub,
		addrBloom,
		outpointBloom,
	)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	account.Idx = idx
	account.ID = id
	account.Name = name
	account.WalletID = walletID
	account.XPub = xPub
	account.AddressBloom = addrBloom
	account.OutpointBloom = outpointBloom
	return account, nil
}

func GetNewAccountIdx(q Querier, walletName string) (uint32, error) {
	row := q.QueryRow("SELECT COALESCE(MAX(idx), -1) + 1 FROM accounts WHERE wallet_id = ?", walletName)
	if row.Err() != nil {
		return 0, errors.WithStack(row.Err())
	}
	var idx uint32
	if err := row.Scan(&idx); err != nil {
		return 0, errors.WithStack(err)
	}
	return idx, nil
}

func GetAccountsForWallet(q Querier, walletName string) ([]*Account, error) {
	rows, err := q.Query(
		"SELECT id, name, wallet_id, idx, change_idx, receiving_idx, xpub, rescan_height, address_bloom, outpoint_bloom FROM accounts WHERE wallet_id = ?",
		walletName,
	)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer rows.Close()
	var out []*Account
	for rows.Next() {
		account := new(Account)
		err := rows.Scan(
			&account.ID,
			&account.Name,
			&account.WalletID,
			&account.Idx,
			&account.ChangeIdx,
			&account.ReceivingIdx,
			&account.XPub,
			&account.RescanHeight,
			&account.AddressBloom,
			&account.OutpointBloom,
		)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		out = append(out, account)
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

func UpdateAccountRecvIdx(tx Transactor, accountID string, idx int) error {
	_, err := tx.Exec(
		"UPDATE accounts SET receiving_idx = ? WHERE id = ?",
		idx,
		accountID,
	)
	return errors.WithStack(err)
}

func UpdateAccountChangeIdx(tx Transactor, accountID string, idx int) error {
	_, err := tx.Exec(
		"UPDATE accounts SET change_idx = ? WHERE id = ?",
		idx,
		accountID,
	)
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
AND (covenant_type = 'NONE' OR covenant_type = 'REDEEM')
AND coins.account_id = ?
AND (coinbase = FALSE OR (coinbase = TRUE AND transactions.block_height <= ?))
`,
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
AND covenant_type = 'NONE' 
AND coins.account_id = ?
AND coinbase = TRUE 
AND transactions.block_height > ?
`,
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
AND covenant_type = 'BID' 
AND coins.account_id = ?
`,
		accountID,
	)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	revealLocked, err := scanBalance(tx, `
SELECT COALESCE(SUM(value), 0) FROM coins
JOIN transactions ON (transactions.hash = coins.tx_hash AND transactions.account_id = coins.account_id)
WHERE spending_tx_hash IS NULL
AND covenant_type = 'REVEAL' 
AND coins.account_id = ?
`,
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
	covenant_type = 'UPDATE'  OR 
	covenant_type = 'REGISTER' OR 
	covenant_type = 'TRANSFER' OR
	covenant_type = 'FINALIZE' OR
	covenant_type = 'REVOKE' 
) 
AND coins.account_id = ?
`,
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
