// Package usecase implements article business logic and defines the repository contract.
package usecase

import (
	"context"
	"fmt"

	"github.com/kun/zhisuo-server/internal/article/entity"
	"github.com/kun/zhisuo-server/internal/port"
)

// ErrUserNotFound is returned when Create is called with a non-existent user ID.
var ErrUserNotFound = port.NewCodedError(port.CodeUserNotFound, "user not found")

// ErrArticleNotFound is returned when an article is not found by ID.
var ErrArticleNotFound = port.NewCodedError(port.CodeArticleNotFound, "article not found")

// ErrArticleVersionConflict is returned on an optimistic-lock version mismatch.
var ErrArticleVersionConflict = port.NewCodedError(port.CodeVersionConflict, "article was updated concurrently")

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

// ListByUser returns a page of articles owned by the given user, newest first.
func (uc *ArticleUseCase) ListByUser(ctx context.Context, userID int64, p port.PageParams) (port.Page, error) {
	articles, total, err := uc.repo.FindByUserID(ctx, userID, p.PageSize, (p.Page-1)*p.PageSize)
	if err != nil {
		return port.Page{}, fmt.Errorf("listing articles: %w", err)
	}

	return port.NewPage(articles, total, p), nil
}

// List returns all articles across all users, ordered by newest first.
func (uc *ArticleUseCase) List(ctx context.Context, p port.PageParams) (port.Page, error) {
	articles, total, err := uc.repo.FindAll(ctx, p.PageSize, (p.Page-1)*p.PageSize)
	if err != nil {
		return port.Page{}, fmt.Errorf("listing articles: %w", err)
	}

	return port.NewPage(articles, total, p), nil
}

// Update replaces the title and content of an existing article identified by id.
// If expectedVersion is non-zero, it enforces optimistic concurrency.
func (uc *ArticleUseCase) Update(ctx context.Context, id, expectedVersion int64, title, content string) (*entity.Article, error) {
	article, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("finding article for update: %w", err)
	}
	if expectedVersion != 0 && article.Version != expectedVersion {
		return nil, ErrArticleVersionConflict
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
