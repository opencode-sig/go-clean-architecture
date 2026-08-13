// Package repository implements the comment persistence layer using MySQL.
package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/kun/zhisuo-server/internal/comment/entity"
	"github.com/kun/zhisuo-server/internal/comment/usecase"
	"github.com/kun/zhisuo-server/internal/port"
	"gorm.io/gorm"
)

// CommentMySQL is the GORM-backed implementation of the comment Repository interface.
type CommentMySQL struct {
	db *gorm.DB
}

// NewCommentMySQL creates a CommentMySQL from a GORM database handle.
func NewCommentMySQL(db *gorm.DB) *CommentMySQL {
	return &CommentMySQL{db: db}
}

// WithTx returns a new repository instance scoped to the given transaction.
func (r *CommentMySQL) WithTx(tx port.Tx) usecase.Repository {
	return &CommentMySQL{db: tx.(*gorm.DB)}
}

// Create inserts a new comment row; GORM populates the ID and timestamps.
func (r *CommentMySQL) Create(ctx context.Context, comment *entity.Comment) error {
	if err := r.db.WithContext(ctx).Create(comment).Error; err != nil {
		return fmt.Errorf("insert comment: %w", err)
	}

	return nil
}

// FindByID retrieves a single comment by its primary key.
func (r *CommentMySQL) FindByID(ctx context.Context, id int64) (*entity.Comment, error) {
	var comment entity.Comment
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&comment).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: %d", usecase.ErrCommentNotFound, id)
		}
		return nil, fmt.Errorf("query comment: %w", err)
	}

	return &comment, nil
}

// FindByArticleID returns a page of comments for the article plus the total count.
func (r *CommentMySQL) FindByArticleID(ctx context.Context, articleID int64, limit, offset int) ([]entity.Comment, int64, error) {
	var comments []entity.Comment
	var total int64

	if err := r.db.WithContext(ctx).
		Model(&entity.Comment{}).
		Where("article_id = ?", articleID).
		Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count comments by article: %w", err)
	}

	if err := r.db.WithContext(ctx).
		Where("article_id = ?", articleID).
		Order("id ASC").
		Limit(limit).
		Offset(offset).
		Find(&comments).Error; err != nil {
		return nil, 0, fmt.Errorf("query comments by article: %w", err)
	}

	return comments, total, nil
}

// Delete removes a comment row by ID. Returns an error if no row was affected.
func (r *CommentMySQL) Delete(ctx context.Context, id int64) error {
	result := r.db.WithContext(ctx).Delete(&entity.Comment{}, id)
	if result.Error != nil {
		return fmt.Errorf("delete comment: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("%w: %d", usecase.ErrCommentNotFound, id)
	}

	return nil
}
