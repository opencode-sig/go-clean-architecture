// Package repository implements the user Repository interface with MySQL.
package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/kun/zhisuo-server/internal/infrastructure"
	"github.com/kun/zhisuo-server/internal/port"
	"github.com/kun/zhisuo-server/internal/user/entity"
	"github.com/kun/zhisuo-server/internal/user/usecase"
)

// UserMySQL implements usecase.Repository backed by a MySQL database.
type UserMySQL struct {
	q infrastructure.Querier
}

// NewUserMySQL creates a UserMySQL backed by a *sql.DB connection pool.
func NewUserMySQL(db *sql.DB) *UserMySQL {
	return &UserMySQL{q: db}
}

// WithTx returns a UserMySQL bound to the given transaction for unit-of-work scoping.
func (r *UserMySQL) WithTx(tx port.Tx) usecase.Repository {
	return &UserMySQL{q: tx.(*sql.Tx)}
}

// Create inserts a new user row and populates the user's ID with the generated value.
func (r *UserMySQL) Create(ctx context.Context, user *entity.User) error {
	result, err := r.q.ExecContext(ctx,
		`INSERT INTO users (username, email, bio, created_at, updated_at) VALUES (?, ?, ?, NOW(), NOW())`,
		user.Username, user.Email, user.Bio,
	)
	if err != nil {
		return fmt.Errorf("insert user: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get last insert id: %w", err)
	}

	user.ID = id
	return nil
}

// FindByID queries a single user by primary key. Returns an error if not found.
func (r *UserMySQL) FindByID(ctx context.Context, id int64) (*entity.User, error) {
	user := &entity.User{}

	err := r.q.QueryRowContext(ctx,
		`SELECT id, username, email, bio, created_at, updated_at FROM users WHERE id = ?`, id,
	).Scan(&user.ID, &user.Username, &user.Email, &user.Bio, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("%w: %d", usecase.ErrUserNotFound, id)
		}
		return nil, fmt.Errorf("query user: %w", err)
	}

	return user, nil
}

// FindAll returns every user row ordered by ID descending.
func (r *UserMySQL) FindAll(ctx context.Context) ([]entity.User, error) {
	rows, err := r.q.QueryContext(ctx,
		`SELECT id, username, email, bio, created_at, updated_at FROM users ORDER BY id DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("query users: %w", err)
	}
	defer rows.Close()

	var users []entity.User
	for rows.Next() {
		var u entity.User
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.Bio, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, u)
	}

	return users, rows.Err()
}

// Update applies new values to an existing user row identified by its ID.
func (r *UserMySQL) Update(ctx context.Context, user *entity.User) error {
	result, err := r.q.ExecContext(ctx,
		`UPDATE users SET username = ?, email = ?, bio = ?, updated_at = NOW() WHERE id = ?`,
		user.Username, user.Email, user.Bio, user.ID,
	)
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}

	if affected == 0 {
		return fmt.Errorf("%w: %d", usecase.ErrUserNotFound, user.ID)
	}

	return nil
}

// Delete removes a user row by primary key. Returns an error if the row does not exist.
func (r *UserMySQL) Delete(ctx context.Context, id int64) error {
	result, err := r.q.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}

	if affected == 0 {
		return fmt.Errorf("%w: %d", usecase.ErrUserNotFound, id)
	}

	return nil
}
