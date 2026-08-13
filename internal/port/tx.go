// Package port defines shared cross-module interfaces for Tx, UserService, and
// ArticleService. Modules depend on these interfaces instead of concrete
// implementations, keeping the architecture decoupled and testable.
package port

import (
	"context"

	"gorm.io/gorm"
)

// Tx is an immutable handle to an active database transaction. It wraps the
// underlying GORM session so callers cannot pass an arbitrary value: the DB is
// only reachable through DB(), which is populated by the infrastructure layer.
type Tx struct {
	db *gorm.DB
}

// NewTx wraps a GORM session as a transaction handle. Only the infrastructure
// layer (TxManager) should create Tx values.
func NewTx(db *gorm.DB) Tx {
	return Tx{db: db}
}

// DB returns the underlying GORM session. It returns nil for a zero Tx, which
// indicates a programmer error — queries fail with a nil DB instead of panicking
// on a type assertion.
func (t Tx) DB() *gorm.DB {
	return t.db
}

// TxManager controls the lifecycle of a database transaction. Use it when you
// need to run multiple repository operations atomically across module boundaries.
// Call Begin to start, then Commit or Rollback to finish; Run wraps all three
// steps so fn is committed only when it returns nil.
type TxManager interface {
	Begin(ctx context.Context) (Tx, error)
	Commit(Tx) error
	Rollback(Tx) error
	Run(ctx context.Context, fn TxFunc) error
}

// TxFunc is the callback signature for TxManager.Run. It receives an open
// transaction that must be passed to repository/service WithTx methods.
type TxFunc func(tx Tx) error
