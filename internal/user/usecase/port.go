// Package usecase contains the business logic for the user module.
package usecase

import (
	"context"

	"github.com/kun/zhisuo-server/internal/port"
	"github.com/kun/zhisuo-server/internal/user/entity"
)

// Repository defines the persistence contract for users.
// The use case layer depends on this interface, not a concrete implementation.
type Repository interface {
	// WithTx returns a new Repository bound to the given transaction.
	WithTx(tx port.Tx) Repository
	// Create persists a new user and populates its ID.
	Create(ctx context.Context, user *entity.User) error
	// FindByID retrieves a single user by ID, or returns an error if not found.
	FindByID(ctx context.Context, id int64) (*entity.User, error)
	// FindAll returns all users ordered by newest first, paged.
	// Returns the page rows and the total count matching the query.
	FindAll(ctx context.Context, limit, offset int) ([]entity.User, int64, error)
	// Update replaces the fields of an existing user identified by its ID.
	Update(ctx context.Context, user *entity.User) error
	// Delete removes a user by ID. Returns an error if the user does not exist.
	Delete(ctx context.Context, id int64) error
}
