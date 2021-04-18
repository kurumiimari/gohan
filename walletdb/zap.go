package walletdb

import "github.com/pkg/errors"

func Zap(tx Transactor, accountID string) error {
	_, err := tx.Exec(`
DELETE FROM coins
JOIN transactions ON transactions.hash = coins.tx_hash
WHERE transactions.block_height = -1 AND
coins.account_id = ?
`,
		accountID,
	)
	if err != nil {
		return errors.WithStack(err)
	}

	_, err = tx.Exec(`
DELETE FROM name_history
JOIN transactions ON transactions.hash = coins.tx_hash
WHERE transactions.block_height = -1 AND
name_history.account_id = ?
`,
		accountID,
	)
	if err != nil {
		return errors.WithStack(err)
	}

	_, err = tx.Exec("DELETE FROM transactions WHERE block_height = -1")
	return errors.WithStack(err)
}
