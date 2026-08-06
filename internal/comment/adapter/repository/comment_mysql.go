// Package repository implements the comment persistence layer using MySQL.
package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/kun/zhisuo-server/internal/comment/entity"
	"github.com/kun/zhisuo-server/internal/comment/usecase"
	"github.com/kun/zhisuo-server/internal/infrastructure"
	"github.com/kun/zhisuo-server/internal/port"
)

// CommentMySQL is the MySQL-backed implementation of the comment Repository interface.
type CommentMySQL struct {
	q infrastructure.Querier
}

// NewCommentMySQL creates a CommentMySQL from a raw database connection.
func NewCommentMySQL(db *sql.DB) *CommentMySQL {
	return &CommentMySQL{q: db}
}

// WithTx returns a new repository instance scoped to the given transaction.
func (r *CommentMySQL) WithTx(tx port.Tx) usecase.Repository {
	return &CommentMySQL{q: tx.(*sql.Tx)}
}

// Create inserts a new comment row and populates the comment's ID field.
func (r *CommentMySQL) Create(ctx context.Context, comment *entity.Comment) error {
	result, err := r.q.ExecContext(ctx,
		`INSERT INTO comments (article_id, user_id, content, created_at, updated_at) VALUES (?, ?, ?, NOW(), NOW())`,
		comment.ArticleID, comment.UserID, comment.Content,
	)
	if err != nil {
		return fmt.Errorf("insert comment: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get last insert id: %w", err)
	}

	comment.ID = id
	return nil
}

// FindByID retrieves a single comment by its primary key.
func (r *CommentMySQL) FindByID(ctx context.Context, id int64) (*entity.Comment, error) {
	comment := &entity.Comment{}

	err := r.q.QueryRowContext(ctx,
		`SELECT id, article_id, user_id, content, created_at, updated_at FROM comments WHERE id = ?`, id,
	).Scan(&comment.ID, &comment.ArticleID, &comment.UserID, &comment.Content, &comment.CreatedAt, &comment.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("%w: %d", usecase.ErrCommentNotFound, id)
		}
		return nil, fmt.Errorf("query comment: %w", err)
	}

	return comment, nil
}

// FindByArticleID returns all comments belonging to the given article, ordered ascending by ID.
func (r *CommentMySQL) FindByArticleID(ctx context.Context, articleID int64) ([]entity.Comment, error) {
	rows, err := r.q.QueryContext(ctx,
		`SELECT id, article_id, user_id, content, created_at, updated_at FROM comments WHERE article_id = ? ORDER BY id ASC`, articleID,
	)
	if err != nil {
		return nil, fmt.Errorf("query comments by article: %w", err)
	}
	defer rows.Close()

	var comments []entity.Comment
	for rows.Next() {
		var c entity.Comment
		if err := rows.Scan(&c.ID, &c.ArticleID, &c.UserID, &c.Content, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan comment: %w", err)
		}
		comments = append(comments, c)
	}

	return comments, rows.Err()
}

// Delete removes a comment row by ID. Returns an error if no row was affected.
func (r *CommentMySQL) Delete(ctx context.Context, id int64) error {
	result, err := r.q.ExecContext(ctx, `DELETE FROM comments WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete comment: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}

	if affected == 0 {
		return fmt.Errorf("%w: %d", usecase.ErrCommentNotFound, id)
	}

	return nil
}
