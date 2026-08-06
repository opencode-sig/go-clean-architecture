// Package usecase implements article business logic and defines the repository contract.
package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/kun/zhisuo-server/internal/article/entity"
	"github.com/kun/zhisuo-server/internal/port"
)

// ErrUserNotFound is returned when Create is called with a non-existent user ID.
var ErrUserNotFound = errors.New("user not found")

// ErrArticleNotFound is returned when an article is not found by ID.
var ErrArticleNotFound = errors.New("article not found")

// ArticleUseCase orchestrates article creation, retrieval, update, and deletion.
type ArticleUseCase struct {
	repo  Repository
	users port.UserService
}

// NewArticleUseCase creates an ArticleUseCase with its repository and user-service dependency.
func NewArticleUseCase(repo Repository, users port.UserService) *ArticleUseCase {
	return &ArticleUseCase{repo: repo, users: users}
}

// Create validates the user exists, then creates and returns a new article.
func (uc *ArticleUseCase) Create(ctx context.Context, userID int64, title, content string) (*entity.Article, error) {
	ok, err := uc.users.Exists(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("checking user: %w", err)
	}
	if !ok {
		return nil, ErrUserNotFound
	}

	article := &entity.Article{
		UserID:  userID,
		Title:   title,
		Content: content,
	}

	if err := uc.repo.Create(ctx, article); err != nil {
		return nil, fmt.Errorf("creating article: %w", err)
	}

	return article, nil
}

// GetByID returns a single article by its ID.
func (uc *ArticleUseCase) GetByID(ctx context.Context, id int64) (*entity.Article, error) {
	article, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("finding article: %w", err)
	}

	return article, nil
}

// ListByUser returns all articles owned by the given user, ordered by newest first.
func (uc *ArticleUseCase) ListByUser(ctx context.Context, userID int64) ([]entity.Article, error) {
	articles, err := uc.repo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("listing articles: %w", err)
	}

	return articles, nil
}

// List returns all articles across all users, ordered by newest first.
func (uc *ArticleUseCase) List(ctx context.Context) ([]entity.Article, error) {
	articles, err := uc.repo.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing articles: %w", err)
	}

	return articles, nil
}

// Update replaces the title and content of an existing article identified by id.
func (uc *ArticleUseCase) Update(ctx context.Context, id int64, title, content string) (*entity.Article, error) {
	article, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("finding article for update: %w", err)
	}

	article.Title = title
	article.Content = content

	if err := uc.repo.Update(ctx, article); err != nil {
		return nil, fmt.Errorf("updating article: %w", err)
	}

	return article, nil
}

// Delete removes an article by its ID.
func (uc *ArticleUseCase) Delete(ctx context.Context, id int64) error {
	if err := uc.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("deleting article: %w", err)
	}

	return nil
}
