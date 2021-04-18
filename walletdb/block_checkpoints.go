package walletdb

import "github.com/pkg/errors"

type BlockCheckpoint struct {
	Height int
	Hash   string
}

func GetBlockCheckpoints(tx Transactor) ([]*BlockCheckpoint, error) {
	rows, err := tx.Query("SELECT height, hash FROM block_checkpoints ORDER BY height DESC")
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer rows.Close()
	var checks []*BlockCheckpoint
	for rows.Next() {
		checkpoint := new(BlockCheckpoint)
		if err := rows.Scan(&checkpoint.Height, &checkpoint.Hash); err != nil {
			return nil, errors.WithStack(err)
		}
		checks = append(checks, checkpoint)
	}
	if rows.Err() != nil {
		return nil, errors.WithStack(err)
	}
	return checks, nil
}

func UpdateBlockCheckpoints(tx Transactor, checkpoints []*BlockCheckpoint) error {
	_, err := tx.Exec("DELETE FROM block_checkpoints")
	if err != nil {
		return errors.WithStack(err)
	}

	for _, check := range checkpoints {
		_, err := tx.Exec(
			"INSERT INTO block_checkpoints(height, hash) VALUES (?, ?)",
			check.Height,
			check.Hash,
		)
		if err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}
