// Package infrastructure provides shared foundational components for all
// business modules: configuration loading, database connectivity, unit-of-work
// transactions, structured logging, Prometheus metrics, and HTTP routing.
package infrastructure

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/kun/zhisuo-server/internal/port"
)

// Querier abstracts *sql.DB and *sql.Tx so repository code works under both normal and transactional contexts.
type Querier interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// TxManager implements port.TxManager with *sql.DB. It manages the begin/commit/rollback lifecycle.
type TxManager struct {
	db *sql.DB
}

// NewTxManager creates a TxManager bound to the given database connection pool.
func NewTxManager(db *sql.DB) *TxManager {
	return &TxManager{db: db}
}

// Begin starts a new database transaction and returns it as a port.Tx.
func (m *TxManager) Begin(ctx context.Context) (port.Tx, error) {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}

	return tx, nil
}

// Commit commits the given transaction.
func (m *TxManager) Commit(tx port.Tx) error {
	return tx.(*sql.Tx).Commit()
}

// Rollback rolls back the given transaction.
func (m *TxManager) Rollback(tx port.Tx) error {
	return tx.(*sql.Tx).Rollback()
}

// TxFunc is the callback signature for TxManager.Run. It receives an open transaction.
type TxFunc = port.TxFunc

// Run begins a transaction, calls fn, and commits on success or rolls back on error.
func (m *TxManager) Run(ctx context.Context, fn TxFunc) error {
	tx, err := m.Begin(ctx)
	if err != nil {
		return err
	}

	if err := fn(tx); err != nil {
		if rbErr := m.Rollback(tx); rbErr != nil {
			return fmt.Errorf("%w: rollback failed: %v", err, rbErr)
		}
		return err
	}

	return m.Commit(tx)
}
