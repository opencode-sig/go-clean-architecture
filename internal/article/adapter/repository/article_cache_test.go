package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kun/zhisuo-server/internal/article/entity"
	"github.com/kun/zhisuo-server/internal/article/usecase"
	"github.com/kun/zhisuo-server/internal/port"
)

// fakeRepo is a minimal in-memory usecase.Repository for decorator tests.
type fakeRepo struct {
	articles map[int64]*entity.Article
	deleted  int64
}

func (f *fakeRepo) WithTx(port.Tx) usecase.Repository { return f }
func (f *fakeRepo) Create(context.Context, *entity.Article) error {
	return nil
}
func (f *fakeRepo) FindByID(_ context.Context, id int64) (*entity.Article, error) {
	if a, ok := f.articles[id]; ok {
		return a, nil
	}
	return nil, usecase.ErrArticleNotFound
}
func (f *fakeRepo) FindByUserID(context.Context, int64) ([]entity.Article, error) {
	return nil, nil
}
func (f *fakeRepo) FindAll(context.Context) ([]entity.Article, error) {
	return nil, nil
}
func (f *fakeRepo) Update(context.Context, *entity.Article) error {
	return nil
}
func (f *fakeRepo) Delete(_ context.Context, id int64) error {
	if _, ok := f.articles[id]; !ok {
		return usecase.ErrArticleNotFound
	}
	delete(f.articles, id)
	f.deleted = id
	return nil
}

// fakeCache records writes and serves a caller-set value to observe hit/miss.
type fakeCache struct {
	data map[string][]byte
}

func (f *fakeCache) Get(_ context.Context, key string) ([]byte, error) {
	if v, ok := f.data[key]; ok {
		return v, nil
	}
	return nil, port.ErrCacheMiss
}
func (f *fakeCache) Set(_ context.Context, key string, value []byte, _ time.Duration) error {
	f.data[key] = value
	return nil
}
func (f *fakeCache) Del(_ context.Context, keys ...string) error {
	for _, k := range keys {
		delete(f.data, k)
	}
	return nil
}

// violates the Repository contract: no nil fakeRepo state
var _ usecase.Repository = (*fakeRepo)(nil)
var _ port.Cache = (*fakeCache)(nil)

func TestArticleCacheHit(t *testing.T) {
	base := &fakeRepo{articles: map[int64]*entity.Article{1: {ID: 1, Title: "T"}}}
	cacheData := make(map[string][]byte)
	c := NewArticleCache(base, &fakeCache{data: cacheData}, time.Minute)

	a, err := c.FindByID(context.Background(), 1)
	if err != nil || a.Title != "T" {
		t.Fatalf("FindByID returned %+v, %v", a, err)
	}

	// Cache should now hold the serialized article.
	if _, ok := cacheData[cacheKey(1)]; !ok {
		t.Fatal("expected cache write on miss")
	}
}

func TestArticleCacheInvalidatesOnDelete(t *testing.T) {
	base := &fakeRepo{articles: map[int64]*entity.Article{1: {ID: 1, Title: "T"}}}
	fake := &fakeCache{data: make(map[string][]byte)}
	c := NewArticleCache(base, fake, time.Minute)

	if _, err := c.FindByID(context.Background(), 1); err != nil {
		t.Fatal(err)
	}
	if err := c.Delete(context.Background(), 1); err != nil {
		t.Fatal(err)
	}
	if len(fake.data) != 0 {
		t.Fatalf("expected key invalidated after delete, got %v", fake.data)
	}
}

func TestArticleCachePenetrationGuardNoRow(t *testing.T) {
	base := &fakeRepo{articles: map[int64]*entity.Article{}}
	fake := &fakeCache{data: make(map[string][]byte)}
	c := NewArticleCache(base, fake, time.Minute)

	if _, err := c.FindByID(context.Background(), 2); !errors.Is(err, usecase.ErrArticleNotFound) {
		t.Fatalf("expected not-found, got %v", err)
	}

	// The miss marker is a zero-length value so the next call is served from cache.
	v, ok := fake.data[cacheKey(2)]
	if !ok || len(v) != 0 {
		t.Fatalf("expected empty miss marker cached, got %q ok=%v", v, ok)
	}
}