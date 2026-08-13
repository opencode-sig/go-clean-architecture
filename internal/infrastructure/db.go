// Package infrastructure provides shared foundational components for all
// business modules: configuration loading, database connectivity, unit-of-work
// transactions, structured logging, Prometheus metrics, and HTTP routing.
package infrastructure

import (
	"context"
	"fmt"

	"github.com/kun/zhisuo-server/internal/port"
	"gorm.io/gorm"
)

// TxManager implements port.TxManager backed by GORM. Begin starts a session
// and wraps it in a typed port.Tx; the same handle is passed to repository
// WithTx methods to bind operations to the transaction.
type TxManager struct {
	db *gorm.DB
}

// NewTxManager creates a TxManager bound to the given GORM database handle.
func NewTxManager(db *gorm.DB) *TxManager {
	return &TxManager{db: db}
}

// Begin starts a new database transaction session.
func (m *TxManager) Begin(ctx context.Context) (port.Tx, error) {
	tx := m.db.WithContext(ctx).Begin()
	if err := tx.Error; err != nil {
		return port.Tx{}, fmt.Errorf("begin tx: %w", err)
	}

	return port.NewTx(tx), nil
}

// Commit commits the given transaction session.
func (m *TxManager) Commit(tx port.Tx) error {
	if err := tx.DB().Commit().Error; err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}

// Rollback rolls back the given transaction session.
func (m *TxManager) Rollback(tx port.Tx) error {
	if err := tx.DB().Rollback().Error; err != nil {
		return fmt.Errorf("rollback tx: %w", err)
	}

	return nil
}

// Run begins a transaction, calls fn, and commits on success or rolls back on error.
func (m *TxManager) Run(ctx context.Context, fn port.TxFunc) error {
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

// compile-time assertion: TxManager implements port.TxManager
var _ port.TxManager = (*TxManager)(nil)
