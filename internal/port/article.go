// Package port defines shared cross-module interfaces (Tx, TxManager,
// UserService, ArticleService) and the unified HTTP response envelope.
// Modules depend on these interfaces instead of concrete implementations,
// keeping the architecture decoupled and testable.
package port

import "context"

// ArticleService is the interface consumed by other modules that need to verify
// article existence or query article state. Implement it in the article module's
// adapter and inject it where cross-module article checks are required.
type ArticleService interface {
	// WithTx returns an ArticleService bound to the given transaction.
	WithTx(tx Tx) ArticleService
	// Exists reports whether the article exists, or an error if the check failed.
	Exists(ctx context.Context, articleID int64) (bool, error)
}
