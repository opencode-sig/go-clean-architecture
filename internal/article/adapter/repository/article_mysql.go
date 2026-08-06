// Package repository provides MySQL implementations of the article repository interface.
package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/kun/zhisuo-server/internal/article/entity"
	"github.com/kun/zhisuo-server/internal/article/usecase"
	"github.com/kun/zhisuo-server/internal/port"
	"gorm.io/gorm"
)

// ArticleMySQL implements usecase.Repository backed by GORM.
type ArticleMySQL struct {
	db *gorm.DB
}

// NewArticleMySQL creates an ArticleMySQL attached to the given GORM database handle.
func NewArticleMySQL(db *gorm.DB) *ArticleMySQL {
	return &ArticleMySQL{db: db}
}

// WithTx returns a new repository bound to the given transaction.
func (r *ArticleMySQL) WithTx(tx port.Tx) usecase.Repository {
	return &ArticleMySQL{db: tx.(*gorm.DB)}
}

// Create inserts an article; GORM populates the ID and timestamps.
func (r *ArticleMySQL) Create(ctx context.Context, article *entity.Article) error {
	if err := r.db.WithContext(ctx).Create(article).Error; err != nil {
		return fmt.Errorf("insert article: %w", err)
	}

	return nil
}

// FindByID returns a single article by its ID, or an error if not found.
func (r *ArticleMySQL) FindByID(ctx context.Context, id int64) (*entity.Article, error) {
	var article entity.Article
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&article).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: %d", usecase.ErrArticleNotFound, id)
		}
		return nil, fmt.Errorf("query article: %w", err)
	}

	return &article, nil
}

// FindByUserID returns all articles owned by the given user, newest first.
func (r *ArticleMySQL) FindByUserID(ctx context.Context, userID int64) ([]entity.Article, error) {
	var articles []entity.Article
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("id DESC").
		Find(&articles).Error; err != nil {
		return nil, fmt.Errorf("query articles by user: %w", err)
	}

	return articles, nil
}

// FindAll returns every article, newest first.
func (r *ArticleMySQL) FindAll(ctx context.Context) ([]entity.Article, error) {
	var articles []entity.Article
	if err := r.db.WithContext(ctx).Order("id DESC").Find(&articles).Error; err != nil {
		return nil, fmt.Errorf("query articles: %w", err)
	}

	return articles, nil
}

// Update persists changes to an existing article. Save performs a full-field UPDATE.
func (r *ArticleMySQL) Update(ctx context.Context, article *entity.Article) error {
	if err := r.db.WithContext(ctx).Save(article).Error; err != nil {
		return fmt.Errorf("update article: %w", err)
	}

	return nil
}

// Delete removes an article by its ID. Returns an error if the article does not exist.
func (r *ArticleMySQL) Delete(ctx context.Context, id int64) error {
	result := r.db.WithContext(ctx).Delete(&entity.Article{}, id)
	if result.Error != nil {
		return fmt.Errorf("delete article: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("%w: %d", usecase.ErrArticleNotFound, id)
	}

	return nil
}
