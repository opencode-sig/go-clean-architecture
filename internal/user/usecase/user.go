// Package usecase contains the business logic for the user module.
package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/kun/zhisuo-server/internal/user/entity"
)

// ErrUserNotFound is returned when a user cannot be found by ID.
var ErrUserNotFound = errors.New("user not found")

// UserUseCase orchestrates user business operations.
// It depends on Repository to abstract persistence.
type UserUseCase struct {
	repo Repository
}

// NewUserUseCase creates a UserUseCase with the given Repository.
func NewUserUseCase(repo Repository) *UserUseCase {
	return &UserUseCase{repo: repo}
}

// Create registers a new user and returns the created entity with its assigned ID.
func (uc *UserUseCase) Create(ctx context.Context, username, email, bio string) (*entity.User, error) {
	user := &entity.User{
		Username: username,
		Email:    email,
		Bio:      bio,
	}

	if err := uc.repo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("creating user: %w", err)
	}

	return user, nil
}

// GetByID retrieves a user by their unique ID.
func (uc *UserUseCase) GetByID(ctx context.Context, id int64) (*entity.User, error) {
	user, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("finding user: %w", err)
	}

	return user, nil
}

// List returns all users, newest first.
func (uc *UserUseCase) List(ctx context.Context) ([]entity.User, error) {
	users, err := uc.repo.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing users: %w", err)
	}

	return users, nil
}

// Update replaces the profile fields of an existing user identified by ID.
func (uc *UserUseCase) Update(ctx context.Context, id int64, username, email, bio string) (*entity.User, error) {
	user, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("finding user for update: %w", err)
	}

	user.Username = username
	user.Email = email
	user.Bio = bio

	if err := uc.repo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("updating user: %w", err)
	}

	return user, nil
}

// Delete removes a user by their ID. Returns an error if the user does not exist.
func (uc *UserUseCase) Delete(ctx context.Context, id int64) error {
	if err := uc.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("deleting user: %w", err)
	}

	return nil
}
