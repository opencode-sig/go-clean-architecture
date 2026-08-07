package cache

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/kun/zhisuo-server/internal/port"
)

// TestRedisBackend exercises Redis against a local server. It is skipped when
// no Redis is reachable at 127.0.0.1:6379, so it stays green in CI without it.
func TestRedisBackend(t *testing.T) {
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("redis not reachable: %v", err)
	}
	defer client.Close()

	c := NewRedis(client)

	if _, err := c.Get(ctx, "miss-key"); !errors.Is(err, port.ErrCacheMiss) {
		t.Fatalf("expected miss, got %v", err)
	}

	const key = "test:redis-cache"
	t.Cleanup(func() { _ = c.Del(context.Background(), key) })

	if err := c.Set(ctx, key, []byte("v"), time.Minute); err != nil {
		t.Fatal(err)
	}

	got, err := c.Get(ctx, key)
	if err != nil || string(got) != "v" {
		t.Fatalf("got %q err %v", got, err)
	}

	if err := c.Del(ctx, key); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Get(ctx, key); !errors.Is(err, port.ErrCacheMiss) {
		t.Fatalf("expected miss after delete, got %v", err)
	}
}
