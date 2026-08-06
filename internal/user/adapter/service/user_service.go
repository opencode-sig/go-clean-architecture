// Package service provides cross-module adapters wrapping user usecase.Repository.
package service

import (
	"context"
	"errors"

	"github.com/kun/zhisuo-server/internal/port"
	"github.com/kun/zhisuo-server/internal/user/usecase"
)

// UserService adapts the user Repository to the port.UserService interface
// for cross-module use, with transaction support.
type UserService struct {
	repo usecase.Repository
}

// NewUserService creates a UserService wrapping the given Repository.
// It returns port.UserService for use by other modules.
func NewUserService(repo usecase.Repository) port.UserService {
	return &UserService{repo: repo}
}

// WithTx returns a UserService bound to the given transaction.
func (s *UserService) WithTx(tx port.Tx) port.UserService {
	return &UserService{repo: s.repo.WithTx(tx)}
}

// Exists reports whether a user with the given ID exists.
// Not-found is returned as (false, nil); database errors are propagated.
func (s *UserService) Exists(ctx context.Context, userID int64) (bool, error) {
	_, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, usecase.ErrUserNotFound) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}
