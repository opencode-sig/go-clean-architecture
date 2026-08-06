// Package repository provides MySQL implementations of the article repository interface.
package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"time"

	"github.com/kun/zhisuo-server/internal/article/entity"
	"github.com/kun/zhisuo-server/internal/article/usecase"
	"github.com/kun/zhisuo-server/internal/port"
)

// ArticleCache is a read-through cache decorator over usecase.Repository.
// It implements the same interface, so it can wrap ArticleMySQL transparently:
//
//	repo := repository.NewArticleCache(repository.NewArticleMySQL(db), cache, ttl)
//
// Cached operations: FindByID (single entity). Mutations invalidate the
// affected keys (Cache-aside). Transactions bypass the cache entirely because
// WithTx delegates to the underlying repository — transactional reads must see
// uncommitted data, which a cache cannot provide.
type ArticleCache struct {
	repo     usecase.Repository
	cache    port.Cache
	ttl      time.Duration
	emptyTTL time.Duration // TTL for "not found" markers (cache-penetration guard)
}

// NewArticleCache wraps base with a cache of the given default TTL.
func NewArticleCache(base usecase.Repository, c port.Cache, ttl time.Duration) *ArticleCache {
	return &ArticleCache{
		repo:     base,
		cache:    c,
		ttl:      ttl,
		emptyTTL: 30 * time.Second,
	}
}

// WithTx bypasses the cache and returns a transactional view of the repository.
func (c *ArticleCache) WithTx(tx port.Tx) usecase.Repository {
	return c.repo.WithTx(tx)
}

// cacheKey returns the storage key for a single article.
func cacheKey(id int64) string {
	return fmt.Sprintf("article:id:%d", id)
}

// Create persists a new article. No cache entry exists yet, so nothing is invalidated.
func (c *ArticleCache) Create(ctx context.Context, article *entity.Article) error {
	return c.repo.Create(ctx, article)
}

// FindByID retrieves an article, populating the cache on a miss.
func (c *ArticleCache) FindByID(ctx context.Context, id int64) (*entity.Article, error) {
	key := cacheKey(id)

	val, err := c.cache.Get(ctx, key)
	switch {
	case err == nil && len(val) == 0:
		// A stored empty value means a previously cached "not found".
		return nil, usecase.ErrArticleNotFound
	case err == nil:
		var a entity.Article
		if json.Unmarshal(val, &a) == nil {
			return &a, nil
		}
	case err != nil && err != port.ErrCacheMiss:
		// Cache backend unhealthy: fall back to the database.
		slog.WarnContext(ctx, "cache get failed, falling back to db", "key", key, "error", err)
	}

	article, err := c.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, usecase.ErrArticleNotFound) {
			// Cache the miss briefly to guard against cache penetration.
			_ = c.cache.Set(ctx, key, nil, c.emptyTTL)
		}
		return nil, err
	}

	data, err := json.Marshal(article)
	if err == nil {
		_ = c.cache.Set(ctx, key, data, c.jitterTTL())
	}

	return article, nil
}

// FindByUserID returns all articles owned by the user. Not cached.
func (c *ArticleCache) FindByUserID(ctx context.Context, userID int64) ([]entity.Article, error) {
	return c.repo.FindByUserID(ctx, userID)
}

// FindAll returns all articles. Not cached.
func (c *ArticleCache) FindAll(ctx context.Context) ([]entity.Article, error) {
	return c.repo.FindAll(ctx)
}

// Update persists changes and invalidates the affected cache key.
func (c *ArticleCache) Update(ctx context.Context, article *entity.Article) error {
	if err := c.repo.Update(ctx, article); err != nil {
		return err
	}

	return c.cache.Del(ctx, cacheKey(article.ID))
}

// Delete removes an article and invalidates its cache key.
func (c *ArticleCache) Delete(ctx context.Context, id int64) error {
	if err := c.repo.Delete(ctx, id); err != nil {
		return err
	}

	return c.cache.Del(ctx, cacheKey(id))
}

// jitterTTL spreads TTLs so keys do not expire simultaneously (cache avalanche).
func (c *ArticleCache) jitterTTL() time.Duration {
	jitter := 0.9 + rand.Float64()*0.2 // 90%..110% of the base TTL
	return time.Duration(float64(c.ttl) * jitter)
}

// compile-time assertion: ArticleCache implements usecase.Repository
var _ usecase.Repository = (*ArticleCache)(nil)