// Package repository provides MySQL implementations of the article repository interface.
package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/kun/zhisuo-server/internal/article/entity"
	"github.com/kun/zhisuo-server/internal/article/usecase"
	"github.com/kun/zhisuo-server/internal/infrastructure"
	"github.com/kun/zhisuo-server/internal/port"
)

// ArticleMySQL implements usecase.Repository backed by a MySQL database.
type ArticleMySQL struct {
	q infrastructure.Querier
}

// NewArticleMySQL creates an ArticleMySQL attached to the given *sql.DB.
func NewArticleMySQL(db *sql.DB) *ArticleMySQL {
	return &ArticleMySQL{q: db}
}

// WithTx returns a new repository bound to the given transaction.
func (r *ArticleMySQL) WithTx(tx port.Tx) usecase.Repository {
	return &ArticleMySQL{q: tx.(*sql.Tx)}
}

// Create inserts an article and populates its ID field.
func (r *ArticleMySQL) Create(ctx context.Context, article *entity.Article) error {
	result, err := r.q.ExecContext(ctx,
		`INSERT INTO articles (user_id, title, content, created_at, updated_at) VALUES (?, ?, ?, NOW(), NOW())`,
		article.UserID, article.Title, article.Content,
	)
	if err != nil {
		return fmt.Errorf("insert article: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get last insert id: %w", err)
	}

	article.ID = id
	return nil
}

// FindByID returns a single article by its ID, or an error if not found.
func (r *ArticleMySQL) FindByID(ctx context.Context, id int64) (*entity.Article, error) {
	article := &entity.Article{}

	err := r.q.QueryRowContext(ctx,
		`SELECT id, user_id, title, content, created_at, updated_at FROM articles WHERE id = ?`, id,
	).Scan(&article.ID, &article.UserID, &article.Title, &article.Content, &article.CreatedAt, &article.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("%w: %d", usecase.ErrArticleNotFound, id)
		}
		return nil, fmt.Errorf("query article: %w", err)
	}

	return article, nil
}

// FindByUserID returns all articles owned by the given user, newest first.
func (r *ArticleMySQL) FindByUserID(ctx context.Context, userID int64) ([]entity.Article, error) {
	rows, err := r.q.QueryContext(ctx,
		`SELECT id, user_id, title, content, created_at, updated_at FROM articles WHERE user_id = ? ORDER BY id DESC`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("query articles by user: %w", err)
	}
	defer rows.Close()

	var articles []entity.Article
	for rows.Next() {
		var a entity.Article
		if err := rows.Scan(&a.ID, &a.UserID, &a.Title, &a.Content, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan article: %w", err)
		}
		articles = append(articles, a)
	}

	return articles, rows.Err()
}

// FindAll returns every article, newest first.
func (r *ArticleMySQL) FindAll(ctx context.Context) ([]entity.Article, error) {
	rows, err := r.q.QueryContext(ctx,
		`SELECT id, user_id, title, content, created_at, updated_at FROM articles ORDER BY id DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("query articles: %w", err)
	}
	defer rows.Close()

	var articles []entity.Article
	for rows.Next() {
		var a entity.Article
		if err := rows.Scan(&a.ID, &a.UserID, &a.Title, &a.Content, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan article: %w", err)
		}
		articles = append(articles, a)
	}

	return articles, rows.Err()
}

// Update persists changes to an existing article's title and content.
func (r *ArticleMySQL) Update(ctx context.Context, article *entity.Article) error {
	result, err := r.q.ExecContext(ctx,
		`UPDATE articles SET title = ?, content = ?, updated_at = NOW() WHERE id = ?`,
		article.Title, article.Content, article.ID,
	)
	if err != nil {
		return fmt.Errorf("update article: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}

	if affected == 0 {
		return fmt.Errorf("%w: %d", usecase.ErrArticleNotFound, article.ID)
	}

	return nil
}

// Delete removes an article by its ID.
func (r *ArticleMySQL) Delete(ctx context.Context, id int64) error {
	result, err := r.q.ExecContext(ctx, `DELETE FROM articles WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete article: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}

	if affected == 0 {
		return fmt.Errorf("%w: %d", usecase.ErrArticleNotFound, id)
	}

	return nil
}
