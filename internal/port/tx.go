// Package port defines shared cross-module interfaces for Tx, UserService, and
// ArticleService. Modules depend on these interfaces instead of concrete
// implementations, keeping the architecture decoupled and testable.
package port

import "context"

// Tx is an opaque token representing an active database transaction. It carries
// no methods — use TxManager to control its lifecycle and pass it to repository
// WithTx methods to bind operations to the same transaction.
type Tx interface{}

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
