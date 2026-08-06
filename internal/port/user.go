// Package port defines shared cross-module interfaces (Tx, TxManager,
// UserService, ArticleService) and the unified HTTP response envelope.
// Modules depend on these interfaces instead of concrete implementations,
// keeping the architecture decoupled and testable.
package port

import "context"

// UserService is the interface consumed by other modules that need to verify
// user existence or query user state. Implement it in the user module's adapter
// and inject it where cross-module user checks are required.
type UserService interface {
	// WithTx returns a UserService bound to the given transaction.
	WithTx(tx Tx) UserService
	// Exists reports whether the user exists, or an error if the check failed.
	Exists(ctx context.Context, userID int64) (bool, error)
}
