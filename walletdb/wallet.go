package walletdb

import (
	"github.com/pkg/errors"
)

type Wallet struct {
	ID        string
	Seed      string
	WatchOnly bool
}

func CreateWallet(tx Transactor, wallet *Wallet) (*Wallet, error) {
	_, err := tx.Exec(`
INSERT INTO wallets (id, seed, watch_only)
VALUES (?, ?, ?)
`,
		wallet.ID,
		wallet.Seed,
		wallet.WatchOnly,
	)
	return wallet, errors.WithStack(err)
}

func GetWallets(tx Transactor) ([]*Wallet, error) {
	rows, err := tx.Query(`
SELECT id, seed, watch_only
FROM wallets ORDER BY id
`)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer rows.Close()
	var out []*Wallet
	for rows.Next() {
		wallet := new(Wallet)
		err := rows.Scan(
			&wallet.ID,
			&wallet.Seed,
			&wallet.WatchOnly,
		)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		out = append(out, wallet)
	}
	return out, errors.WithStack(rows.Err())
}
