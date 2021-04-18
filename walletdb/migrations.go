package walletdb

import (
	"github.com/kurumiimari/gohan/log"
	"github.com/pkg/errors"
	"time"
)

var logger = log.ModuleLogger("migrations")

const CreateMigrationsQuery = `
CREATE TABLE IF NOT EXISTS migrations (
	id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
	name VARCHAR NOT NULL,
	applied_at INTEGER NOT NULL
);
`

type Migration struct {
	Query string
	Name  string
}

var Migrations = []*Migration{
	{
		Query: `
CREATE TABLE wallets (
	id VARCHAR NOT NULL PRIMARY KEY,
	seed VARCHAR NOT NULL,
	watch_only BOOLEAN NOT NULL
);
`,
		Name: "create_wallets",
	},
	{
		Query: `
CREATE TABLE accounts (
	id VARCHAR NOT NULL PRIMARY KEY,
	name VARCHAR NOT NULL,
	wallet_id VARCHAR NOT NULL,
	idx INTEGER NOT NULL,
	change_idx INTEGER NOT NULL DEFAULT 0,
	receiving_idx INTEGER NOT NULL DEFAULT 0,
	xpub VARCHAR(111) NOT NULL,
	rescan_height INTEGER NOT NULL DEFAULT 0,
    address_bloom BLOB NOT NULL,
	outpoint_bloom BLOB NOT NULL,
	FOREIGN KEY (wallet_id) REFERENCES wallets(id)
);
`,
		Name: "create_accounts",
	},
	{
		Query: `
CREATE TABLE transactions (
	id INTEGER NOT NULL PRIMARY KEY,
	account_id VARCHAR NOT NULL,
	hash VARCHAR(64) NOT NULL,
	idx INTEGER NOT NULL,
	block_height INTEGER NOT NULL,
	block_hash VARCHAR(64) NOT NULL,
	raw BLOB NOT NULL,
	time INTEGER NOT NULL
);

CREATE UNIQUE INDEX idx_unique_account_id_hash ON transactions(account_id, hash);
`,
		Name: "create_transactions",
	},
	{
		Query: `
CREATE TABLE addresses (
	address VARCHAR NOT NULL PRIMARY KEY,
	account_id VARCHAR NOT NULL,
	branch INTEGER NOT NULL,
	idx INTEGER NOT NULL,
	FOREIGN KEY (account_id) REFERENCES accounts(id)
);
`,
		Name: "create_addresses",
	},
	{
		Query: `
CREATE TABLE coins (
	id INTEGER NOT NULL PRIMARY KEY,
	account_id VARCHAR NOT NULL,
	tx_hash VARCHAR(64) NOT NULL,
	out_idx INTEGER NOT NULL,
	value INTEGER NOT NULL,
	address VARCHAR NOT NULL,
	coinbase BOOLEAN NOT NULL,
	covenant_type VARCHAR NOT NULL,
	covenant_items BLOB,
	spending_tx_hash VARCHAR(64),
	FOREIGN KEY (account_id) REFERENCES accounts(id),
	FOREIGN KEY (tx_hash) REFERENCES transactions(hash),
	FOREIGN KEY (spending_tx_hash) REFERENCES transactions(hash),
	FOREIGN KEY (address) REFERENCES addresses(address)
);

CREATE INDEX idx_coins_tx_hash ON coins(tx_hash);
`,
		Name: "create_coins",
	},
	{
		Query: `
CREATE TABLE names (
	id INTEGER NOT NULL PRIMARY KEY,
	account_id VARCHAR NOT NULL,
    name VARCHAR NOT NULL,
	hash VARCHAR(64) NOT NULL,
	status VARCHAR NOT NULL,
	FOREIGN KEY (account_id) REFERENCES accounts(id)
);

CREATE INDEX idx_names_account_id ON names(account_id);
CREATE INDEX idx_names_name ON names(name);
CREATE INDEX idx_names_hash ON names(hash);
CREATE UNIQUE INDEX idx_uniq_names_account_id_name ON names(account_id, name);
CREATE UNIQUE INDEX idx_uniq_names_account_id_hash ON names(account_id, hash);
`,
		Name: "create_names",
	},
	{
		Query: `
CREATE TABLE name_history (
	id INTEGER NOT NULL PRIMARY KEY,
	account_id VARCHAR NOT NULL, 
	name VARCHAR NOT NULL,
	type VARCHAR NOT NULL,
	tx_hash VARCHAR NOT NULL,
	out_idx INTEGER NOT NULL,
	value INTEGER NOT NULL,
	bid_value INTEGER,
	parent_tx_hash VARCHAR,
	parent_out_idx INTEGER,
	FOREIGN KEY (account_id) REFERENCES accounts(account_id)
);

CREATE INDEX idx_name_history_account_id_name ON name_history(account_id, name);
CREATE UNIQUE INDEX idx_uniq_name_history_outpoint ON name_history(account_id, tx_hash, out_idx);
`,
		Name: "create_name_history",
	},
	{
		Query:`
CREATE TABLE block_checkpoints (
	height INTEGER NOT NULL PRIMARY KEY,
	hash VARCHAR NOT NULL
);

CREATE INDEX idx_block_checkpoints_hash ON block_checkpoints(hash);
`,
		Name: "create_block_checkpoints",
	},
}

func MigrateDB(engine *Engine) error {
	return engine.Transaction(func(tx Transactor) error {
		logger.Debug("creating migrations table")
		_, err := tx.Exec(CreateMigrationsQuery)
		if err != nil {
			return errors.WithStack(err)
		}

		migRow := tx.QueryRow("SELECT COALESCE(MAX(id), 0) FROM migrations")
		if migRow.Err() != nil {
			return errors.WithStack(migRow.Err())
		}
		var latestMigID int
		if err := migRow.Scan(&latestMigID); err != nil {
			return errors.WithStack(err)
		}

		if latestMigID == len(Migrations) {
			logger.Info("migrations up to date")
			return nil
		}

		logger.Info("running migrations")
		for i := latestMigID; i < len(Migrations); i++ {
			mig := Migrations[i]
			logger.Debug("executing migration", "name", mig.Name, "version", i)
			if err := ExecMigration(tx, mig); err != nil {
				return err
			}
		}
		logger.Info("successfully migrated database")
		return nil
	})
}

func ExecMigration(tx Transactor, migration *Migration) error {
	if _, err := tx.Exec(migration.Query); err != nil {
		return errors.Wrapf(err, "error executing migration %s", migration.Name)
	}
	_, err := tx.Exec(
		"INSERT INTO migrations (name, applied_at) VALUES (?, ?)",
		migration.Name,
		time.Now().Unix(),
	)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}
