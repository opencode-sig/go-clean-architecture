// Package port defines shared cross-module interfaces (Tx, TxManager,
// Cache, UserService, ArticleService) and the unified HTTP response envelope.
// Modules depend on these interfaces instead of concrete implementations,
// keeping the architecture decoupled and testable.
package port

import (
	"context"
	"errors"
	"time"
)

// ErrCacheMiss is returned by Cache.Get when the given key has no value.
// It is a sentinel error; implementers must return it for a cache miss and
// must not treat a miss as a failure of the underlying cache system.
var ErrCacheMiss = errors.New("cache miss")

// Cache is a byte-oriented key-value cache abstraction. Implementations are
// pure infrastructure (in-memory, Redis) and never see domain types — the
// caller (repository decorator) is responsible for (de)serialization.
//
// Contract:
//   - Get returns ErrCacheMiss for a miss; other errors mean the cache backend
//     is unhealthy, and callers should fall back to the database.
//   - Set with ttl <= 0 means "no expiration"; backends may still evict.
//   - Del is idempotent: deleting a non-existent key is not an error.
type Cache interface {
	// Get returns the value for key, or ErrCacheMiss if absent.
	Get(ctx context.Context, key string) ([]byte, error)
	// Set stores value for key with the given TTL.
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	// Del removes the given keys.
	Del(ctx context.Context, keys ...string) error
}

// CacheListInvalidator is an optional capability implemented by cache backends
// that can delete all keys with a given prefix. List caches are keyed by
// page/size, so writes must invalidate them by prefix. Backends without this
// capability degrade to short TTLs.
type CacheListInvalidator interface {
	DeleteByPrefix(ctx context.Context, prefix string) error
}
