package walletdb

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
	"path"
	"sync"
)

type Engine struct {
	db  *sql.DB
	mtx sync.Mutex
}

type Scanner interface {
	Scan(dest ...interface{}) error
}

type Querier interface {
	Query(q string, args ...interface{}) (*sql.Rows, error)
	QueryRow(q string, args ...interface{}) *sql.Row
	Exec(q string, args ...interface{}) (sql.Result, error)
}

type Transactor interface {
	Querier
}

func NewEngine(dbPath string) (*Engine, error) {
	db, err := sql.Open("sqlite3", path.Join(dbPath, "wallet.db"))
	if err != nil {
		return nil, errors.Wrap(err, "error opening DB")
	}
	return &Engine{
		db: db,
	}, nil
}

func (e *Engine) Transaction(cb func(tx Transactor) error) error {
	e.mtx.Lock()
	defer e.mtx.Unlock()
	tx, err := e.db.Begin()
	if err != nil {
		panic(err)
	}

	childTx := &transactor{tx: tx}
	if err := cb(childTx); err != nil {
		cbErr := err
		if err := tx.Rollback(); err != nil {
			panic("error rolling back transaction!")
		}
		return cbErr
	}

	if err := tx.Commit(); err != nil {
		panic(err)
	}

	return nil
}

type transactor struct {
	tx *sql.Tx
}

func (t transactor) Query(q string, args ...interface{}) (*sql.Rows, error) {
	return t.tx.Query(q, args...)
}

func (t transactor) QueryRow(q string, args ...interface{}) *sql.Row {
	return t.tx.QueryRow(q, args...)
}

func (t transactor) Exec(q string, args ...interface{}) (sql.Result, error) {
	return t.tx.Exec(q, args...)
}
