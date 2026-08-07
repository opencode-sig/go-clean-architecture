// Package usecase defines the repository contract and business logic for articles.
package usecase

import (
	"context"

	"github.com/kun/zhisuo-server/internal/article/entity"
	"github.com/kun/zhisuo-server/internal/port"
)

// Repository defines the data-access contract the use case depends on.
type Repository interface {
	// WithTx returns a new Repository bound to the given transaction.
	WithTx(tx port.Tx) Repository
	// Create persists a new article and populates its ID.
	Create(ctx context.Context, article *entity.Article) error
	// FindByID retrieves a single article by ID, or returns an error if not found.
	FindByID(ctx context.Context, id int64) (*entity.Article, error)
	// FindByUserID returns all articles owned by the given user, newest first, paged.
	FindByUserID(ctx context.Context, userID int64, limit, offset int) ([]entity.Article, int64, error)
	// FindAll returns all articles ordered by newest first, paged.
	FindAll(ctx context.Context, limit, offset int) ([]entity.Article, int64, error)
	// Update replaces the fields of an existing article identified by its ID.
	Update(ctx context.Context, article *entity.Article) error
	// Delete removes an article by ID. Returns an error if the article does not exist.
	Delete(ctx context.Context, id int64) error
}
