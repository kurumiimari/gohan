package walletdb

import (
	"github.com/pkg/errors"
)

type Address struct {
	Address   string
	AccountID string
	Branch    int
	Idx       int
}

func CreateAddress(tx Transactor, address, accountID string, branch, idx int) (*Address, error) {
	addr := new(Address)
	_, err := tx.Exec(`
INSERT INTO addresses (address, account_id, branch, idx)
VALUES (?, ?, ?, ?);
`,
		address,
		accountID,
		branch,
		idx,
	)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	addr.Address = address
	addr.AccountID = accountID
	addr.Branch = branch
	addr.Idx = idx
	return addr, nil
}

func GetAddress(q Querier, accountID, address string) (*Address, error) {
	row := q.QueryRow(
		"SELECT branch, idx FROM addresses WHERE account_id = ? AND address = ?",
		accountID,
		address,
	)
	if row.Err() != nil {
		return nil, errors.WithStack(row.Err())
	}
	addr := &Address{
		Address:   address,
		AccountID: accountID,
	}
	if err := row.Scan(&addr.Branch, &addr.Idx); err != nil {
		return nil, errors.WithStack(err)
	}
	return addr, nil
}

func GetLookaheadTips(q Querier, accountID string) ([2]uint32, error) {
	var tips [2]uint32
	rows, err := q.Query(
		"SELECT COALESCE(MAX(idx), -1) FROM addresses WHERE account_id = ? GROUP BY branch ORDER BY branch",
		accountID,
	)
	if err != nil {
		return tips, errors.WithStack(err)
	}
	defer rows.Close()
	for i := 0; i < 2; i++ {
		rows.Next()
		if err := rows.Scan(&tips[i]); err != nil {
			return tips, errors.WithStack(err)
		}
	}
	return tips, errors.WithStack(rows.Err())
}
