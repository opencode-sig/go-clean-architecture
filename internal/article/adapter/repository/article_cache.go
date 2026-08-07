// Package repository provides MySQL implementations of the article repository interface.
package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"strconv"
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
// Cached operations: FindByID (single entity) and list reads (short TTL).
// Mutations invalidate the affected keys (Cache-aside). Transactions bypass the
// cache entirely because WithTx delegates to the underlying repository —
// transactional reads must see uncommitted data, which a cache cannot provide.
type ArticleCache struct {
	repo     usecase.Repository
	cache    port.Cache
	ttl      time.Duration
	emptyTTL time.Duration // TTL for "not found" markers (cache-penetration guard)
	listTTL  time.Duration // shorter TTL for paginated list reads
}

// NewArticleCache wraps base with a cache of the given default TTL.
func NewArticleCache(base usecase.Repository, c port.Cache, ttl time.Duration) *ArticleCache {
	listTTL := ttl / 5
	if listTTL < 15*time.Second {
		listTTL = 15 * time.Second
	}
	return &ArticleCache{
		repo:     base,
		cache:    c,
		ttl:      ttl,
		emptyTTL: 30 * time.Second,
		listTTL:  listTTL,
	}
}

// WithTx bypasses the cache and returns a transactional view of the repository.
func (c *ArticleCache) WithTx(tx port.Tx) usecase.Repository {
	return c.repo.WithTx(tx)
}

func cacheKey(id int64) string {
	return fmt.Sprintf("article:id:%d", id)
}

// Create persists a new article and invalidates list caches that may include it.
func (c *ArticleCache) Create(ctx context.Context, article *entity.Article) error {
	if err := c.repo.Create(ctx, article); err != nil {
		return err
	}
	return c.invalidateLists(ctx)
}

// FindByID retrieves an article, populating the cache on a miss.
func (c *ArticleCache) FindByID(ctx context.Context, id int64) (*entity.Article, error) {
	key := cacheKey(id)

	val, err := c.cache.Get(ctx, key)
	switch {
	case err == nil && len(val) == 0:
		// A stored empty value means a previously cached "not found".
		return nil, usecase.ErrArticleNotFound
	case err == nil && len(val) > 0:
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

// FindByUserID returns a page of articles owned by the user, caching list reads briefly.
func (c *ArticleCache) FindByUserID(ctx context.Context, userID int64, limit, offset int) ([]entity.Article, int64, error) {
	return c.cachedList(ctx, listKeyUser(userID, limit, offset), limit, offset, func() ([]entity.Article, int64, error) {
		return c.repo.FindByUserID(ctx, userID, limit, offset)
	})
}

// FindAll returns a page of articles, cached briefly.
func (c *ArticleCache) FindAll(ctx context.Context, limit, offset int) ([]entity.Article, int64, error) {
	return c.cachedList(ctx, listKeyAll(limit, offset), limit, offset, func() ([]entity.Article, int64, error) {
		return c.repo.FindAll(ctx, limit, offset)
	})
}

func listKeyUser(userID int64, limit, offset int) string {
	return cacheListKey("user", userID, limit, offset)
}

func listKeyAll(limit, offset int) string {
	return cacheListKey("all", 0, limit, offset)
}

func cacheListKey(scope string, scopeID int64, limit, offset int) string {
	return "article:list:" + scope + ":" + strconv.FormatInt(scopeID, 10) + ":" +
		strconv.Itoa(limit) + ":" + strconv.Itoa(offset)
}

// cachedList implements cache-aside paging: it caches the full page+total tuple
// with a short TTL (listTTL). An empty byte slice marks a cached empty list.
func (c *ArticleCache) cachedList(ctx context.Context, key string, limit, offset int, loader func() ([]entity.Article, int64, error)) ([]entity.Article, int64, error) {
	if val, err := c.cache.Get(ctx, key); err == nil {
		var page struct {
			Items []entity.Article `json:"items"`
			Total int64            `json:"total"`
		}
		if json.Unmarshal(val, &page) == nil {
			if page.Items == nil {
				page.Items = []entity.Article{}
			}
			return page.Items, page.Total, nil
		}
	} else if err != port.ErrCacheMiss {
		slog.WarnContext(ctx, "cache list get failed, falling back to db", "key", key, "error", err)
	}

	items, total, err := loader()
	if err != nil {
		return nil, 0, err
	}

	if data, err := json.Marshal(struct {
		Items []entity.Article `json:"items"`
		Total int64            `json:"total"`
	}{Items: items, Total: total}); err == nil {
		_ = c.cache.Set(ctx, key, data, c.jitterListTTL())
	}

	return items, total, nil
}

// Update persists changes, invalidates the entity key, then invalidates lists.
func (c *ArticleCache) Update(ctx context.Context, article *entity.Article) error {
	if err := c.repo.Update(ctx, article); err != nil {
		return err
	}
	_ = c.cache.Del(ctx, cacheKey(article.ID))
	return c.invalidateLists(ctx)
}

// Delete removes an article and invalidates its cache keys.
func (c *ArticleCache) Delete(ctx context.Context, id int64) error {
	if err := c.repo.Delete(ctx, id); err != nil {
		return err
	}
	_ = c.cache.Del(ctx, cacheKey(id))
	return c.invalidateLists(ctx)
}

// invalidateLists clears paginated list caches via the cache backend's PatternDel.
func (c *ArticleCache) invalidateLists(ctx context.Context) error {
	pd, ok := c.cache.(port.CacheListInvalidator)
	if !ok {
		slog.WarnContext(ctx, "cache backend does not support pattern delete; list caches rely on short TTL")
		return nil
	}
	return pd.DeleteByPrefix(ctx, "article:list:")
}

// jitterTTL spreads single-entity TTLs (cache avalanche guard).
func (c *ArticleCache) jitterTTL() time.Duration {
	return jitter(c.ttl)
}

// jitterListTTL spreads the list TTL.
func (c *ArticleCache) jitterListTTL() time.Duration {
	return jitter(c.listTTL)
}

func jitter(base time.Duration) time.Duration {
	return time.Duration(float64(base) * (0.9 + rand.Float64()*0.2))
}

// compile-time assertion: ArticleCache implements usecase.Repository
var _ usecase.Repository = (*ArticleCache)(nil)
