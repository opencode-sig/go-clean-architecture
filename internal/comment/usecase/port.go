// Package usecase defines the persistence contract and business logic for comments.
package usecase

import (
	"context"

	"github.com/kun/zhisuo-server/internal/comment/entity"
	"github.com/kun/zhisuo-server/internal/port"
)

// Repository defines the persistence contract for comments.
// The use case depends on this interface, not a concrete implementation.
type Repository interface {
	// WithTx returns a new Repository bound to the given transaction.
	WithTx(tx port.Tx) Repository
	// Create persists a new comment and populates its ID.
	Create(ctx context.Context, comment *entity.Comment) error
	// FindByID retrieves a single comment by ID, or returns an error if not found.
	FindByID(ctx context.Context, id int64) (*entity.Comment, error)
	// FindByArticleID returns a page of comments on the given article plus total count.
	FindByArticleID(ctx context.Context, articleID int64, limit, offset int) ([]entity.Comment, int64, error)
	// Delete removes a comment by ID. Returns an error if the comment does not exist.
	Delete(ctx context.Context, id int64) error
}
