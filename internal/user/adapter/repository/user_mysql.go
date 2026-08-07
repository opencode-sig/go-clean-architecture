// Package repository implements the user Repository interface with MySQL.
package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/kun/zhisuo-server/internal/port"
	"github.com/kun/zhisuo-server/internal/user/entity"
	"github.com/kun/zhisuo-server/internal/user/usecase"
	"gorm.io/gorm"
)

// UserMySQL implements usecase.Repository backed by GORM.
type UserMySQL struct {
	db *gorm.DB
}

// NewUserMySQL creates a UserMySQL backed by a GORM database handle.
func NewUserMySQL(db *gorm.DB) *UserMySQL {
	return &UserMySQL{db: db}
}

// WithTx returns a UserMySQL bound to the given transaction for unit-of-work scoping.
func (r *UserMySQL) WithTx(tx port.Tx) usecase.Repository {
	return &UserMySQL{db: tx.(*gorm.DB)}
}

// Create inserts a new user row; GORM populates the ID and timestamps.
func (r *UserMySQL) Create(ctx context.Context, user *entity.User) error {
	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		return fmt.Errorf("insert user: %w", err)
	}

	return nil
}

// FindByID queries a single user by primary key. Returns an error if not found.
func (r *UserMySQL) FindByID(ctx context.Context, id int64) (*entity.User, error) {
	var user entity.User
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: %d", usecase.ErrUserNotFound, id)
		}
		return nil, fmt.Errorf("query user: %w", err)
	}

	return &user, nil
}

// FindAll returns a page of users ordered by ID descending plus the total count.
func (r *UserMySQL) FindAll(ctx context.Context, limit, offset int) ([]entity.User, int64, error) {
	var users []entity.User
	var total int64

	if err := r.db.WithContext(ctx).
		Model(&entity.User{}).
		Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count users: %w", err)
	}

	if err := r.db.WithContext(ctx).
		Order("id DESC").
		Limit(limit).
		Offset(offset).
		Find(&users).Error; err != nil {
		return nil, 0, fmt.Errorf("query users: %w", err)
	}

	return users, total, nil
}

// Update persists all fields of the user with optimistic locking on Version.
func (r *UserMySQL) Update(ctx context.Context, user *entity.User) error {
	result := r.db.WithContext(ctx).
		Model(&entity.User{}).
		Where("id = ? AND version = ?", user.ID, user.Version).
		UpdateColumns(map[string]any{
			"username":   user.Username,
			"email":      user.Email,
			"bio":        user.Bio,
			"version":    gorm.Expr("version + 1"),
			"updated_at": gorm.Expr("NOW()"),
		})
	if result.Error != nil {
		return fmt.Errorf("update user: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("%w: %d", usecase.ErrUserVersionConflict, user.ID)
	}

	user.Version++

	return nil
}

// Delete removes a user row by primary key. Returns an error if the row does not exist.
func (r *UserMySQL) Delete(ctx context.Context, id int64) error {
	result := r.db.WithContext(ctx).Delete(&entity.User{}, id)
	if result.Error != nil {
		return fmt.Errorf("delete user: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("%w: %d", usecase.ErrUserNotFound, id)
	}

	return nil
}
