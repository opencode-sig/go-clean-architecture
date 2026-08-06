// Package service provides cross-module adapters that expose article operations via port.ArticleService.
package service

import (
	"context"
	"errors"

	"github.com/kun/zhisuo-server/internal/article/usecase"
	"github.com/kun/zhisuo-server/internal/port"
)

// ArticleService adapts the article repository to the port.ArticleService interface.
type ArticleService struct {
	repo usecase.Repository
}

// NewArticleService returns a port.ArticleService backed by the article repository.
func NewArticleService(repo usecase.Repository) port.ArticleService {
	return &ArticleService{repo: repo}
}

// WithTx returns a new service bound to the given transaction.
func (s *ArticleService) WithTx(tx port.Tx) port.ArticleService {
	return &ArticleService{repo: s.repo.WithTx(tx)}
}

// Exists reports whether an article with the given ID exists.
// Not-found is returned as (false, nil); database errors are propagated.
func (s *ArticleService) Exists(ctx context.Context, articleID int64) (bool, error) {
	_, err := s.repo.FindByID(ctx, articleID)
	if err != nil {
		if errors.Is(err, usecase.ErrArticleNotFound) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}
