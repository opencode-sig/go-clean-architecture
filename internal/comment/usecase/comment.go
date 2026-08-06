// Package usecase implements the business logic for comment operations.
package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/kun/zhisuo-server/internal/comment/entity"
	"github.com/kun/zhisuo-server/internal/port"
)

var (
	// ErrUserNotFound is returned when the comment author's user ID does not exist.
	ErrUserNotFound = errors.New("user not found")
	// ErrArticleNotFound is returned when the target article ID does not exist.
	ErrArticleNotFound = errors.New("article not found")
	// ErrCommentNotFound is returned when a comment cannot be found by ID.
	ErrCommentNotFound = errors.New("comment not found")
)

// CommentUseCase orchestrates comment creation, listing, and deletion.
// It validates that the referenced user and article exist before persisting.
type CommentUseCase struct {
	repo      Repository
	txManager port.TxManager
	articles  port.ArticleService
	users     port.UserService
}

// NewCommentUseCase creates a CommentUseCase with its required dependencies.
func NewCommentUseCase(repo Repository, txManager port.TxManager, articles port.ArticleService, users port.UserService) *CommentUseCase {
	return &CommentUseCase{repo: repo, txManager: txManager, articles: articles, users: users}
}

// Create adds a new comment after verifying the article and author both exist.
// The existence checks and the insert run in one transaction so a concurrent
// delete cannot leave a dangling comment. This is the reference example for
// cross-module transactions: bind services/repositories with WithTx(tx).
func (uc *CommentUseCase) Create(ctx context.Context, articleID, userID int64, content string) (*entity.Comment, error) {
	var comment *entity.Comment

	err := uc.txManager.Run(ctx, func(tx port.Tx) error {
		users := uc.users.WithTx(tx)
		articles := uc.articles.WithTx(tx)
		repo := uc.repo.WithTx(tx)

		ok, err := users.Exists(ctx, userID)
		if err != nil {
			return fmt.Errorf("checking user: %w", err)
		}
		if !ok {
			return ErrUserNotFound
		}

		ok, err = articles.Exists(ctx, articleID)
		if err != nil {
			return fmt.Errorf("checking article: %w", err)
		}
		if !ok {
			return ErrArticleNotFound
		}

		comment = &entity.Comment{
			ArticleID: articleID,
			UserID:    userID,
			Content:   content,
		}

		if err := repo.Create(ctx, comment); err != nil {
			return fmt.Errorf("creating comment: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return comment, nil
}

// ListByArticle returns all comments for a given article, ordered by creation time.
func (uc *CommentUseCase) ListByArticle(ctx context.Context, articleID int64) ([]entity.Comment, error) {
	comments, err := uc.repo.FindByArticleID(ctx, articleID)
	if err != nil {
		return nil, fmt.Errorf("listing comments: %w", err)
	}

	return comments, nil
}

// Delete removes a comment by its ID. Returns an error if the comment does not exist.
func (uc *CommentUseCase) Delete(ctx context.Context, id int64) error {
	if err := uc.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("deleting comment: %w", err)
	}

	return nil
}
