package cache

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/kun/zhisuo-server/internal/port"
)

// Redis implements port.Cache backed by a Redis server. Values are stored as
// raw bytes under the given key; TTL is honored via Redis expiration.
// It accepts a redis.UniversalClient, so both single-node (*redis.Client)
// and cluster (*redis.ClusterClient) topologies are supported.
type Redis struct {
	client redis.UniversalClient
	ttl    time.Duration // default TTL when Set passes ttl <= 0; zero disables
}

// NewRedis creates a Redis cache from an existing redis client.
func NewRedis(client redis.UniversalClient) *Redis {
	return &Redis{client: client}
}

// Ping verifies connectivity to the Redis server.
func (r *Redis) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// Get implements port.Cache. It returns port.ErrCacheMiss when the key does
// not exist (Redis returns redis.Nil).
func (r *Redis) Get(ctx context.Context, key string) ([]byte, error) {
	val, err := r.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, port.ErrCacheMiss
	}

	return val, err
}

// Set implements port.Cache.
func (r *Redis) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if ttl <= 0 {
		ttl = r.ttl
	}
	if ttl <= 0 {
		// Set without expiration (keep TTL).
		return r.client.Set(ctx, key, value, 0).Err()
	}

	return r.client.Set(ctx, key, value, ttl).Err()
}

// Del implements port.Cache. Deleting a non-existent key is a no-op.
func (r *Redis) Del(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}

	return r.client.Del(ctx, keys...).Err()
}

// DeleteByPrefix removes all keys matching the given glob prefix via SCAN.
func (r *Redis) DeleteByPrefix(ctx context.Context, prefix string) error {
	iter := r.client.Scan(ctx, 0, prefix+"*", 100).Iterator()
	var batch []string
	for iter.Next(ctx) {
		batch = append(batch, iter.Val())
		if len(batch) >= 100 {
			if err := r.client.Del(ctx, batch...).Err(); err != nil {
				return err
			}
			batch = nil
		}
	}
	if err := iter.Err(); err != nil {
		return err
	}
	if len(batch) > 0 {
		return r.client.Del(ctx, batch...).Err()
	}

	return nil
}

// Close releases the underlying Redis connection pool.
func (r *Redis) Close() error {
	return r.client.Close()
}

// compile-time assertions: Redis implements port.Cache and CacheListInvalidator
var _ port.Cache = (*Redis)(nil)
var _ port.CacheListInvalidator = (*Redis)(nil)
