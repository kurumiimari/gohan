package walletdb

import (
	"github.com/kurumiimari/gohan/chain"
	"github.com/kurumiimari/gohan/shakedex"
	"github.com/pkg/errors"
)

type Address struct {
	AccountID  string
	Address    *chain.Address
	Derivation chain.Derivation
}

func CreateAddress(tx Transactor, accountID string, address *chain.Address, branch, idx uint32) (*Address, error) {
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
	addr.Derivation = chain.Derivation{branch, idx}
	return addr, nil
}

func GetAddress(q Querier, accountID string, address *chain.Address) (*Address, error) {
	row := q.QueryRow(
		"SELECT branch, idx FROM addresses WHERE account_id = ? AND address = ?",
		accountID,
		address,
	)
	if row.Err() != nil {
		return nil, errors.WithStack(row.Err())
	}
	var branch uint32
	var index uint32
	if err := row.Scan(&branch, &index); err != nil {
		return nil, errors.WithStack(err)
	}
	return &Address{
		Address:    address,
		AccountID:  accountID,
		Derivation: chain.Derivation{branch, index},
	}, nil
}

func GetLookaheadTips(q Querier, accountID string) (map[uint32]uint32, error) {
	tips := make(map[uint32]uint32)
	branches := []uint32{chain.ReceiveBranch, chain.ChangeBranch, shakedex.AddressBranch}
	for _, branch := range branches {
		row := q.QueryRow(
			"SELECT COALESCE(MAX(idx), -1) FROM addresses WHERE account_id = ? AND branch = ?",
			accountID,
			branch,
		)
		if row.Err() != nil {
			return nil, errors.WithStack(row.Err())
		}
		var tip uint32
		if err := row.Scan(&tip); err != nil {
			return nil, errors.WithStack(err)
		}
		tips[branch] = tip
	}
	return tips, nil
}
