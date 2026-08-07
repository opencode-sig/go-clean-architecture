package cache

import (
	"fmt"

	"github.com/redis/go-redis/v9"

	"github.com/kun/zhisuo-server/internal/infrastructure"
	"github.com/kun/zhisuo-server/internal/port"
)

// NewCache returns a port.Cache implementation selected from cfg.CacheType.
// Supported values: "memory" (default) and "redis".
func NewCache(cfg *infrastructure.Config) port.Cache {
	switch cfg.CacheType {
	case "redis":
		client := redis.NewClient(&redis.Options{
			Addr:     fmt.Sprintf("%s:%d", cfg.RedisHost, cfg.RedisPort),
			DB:       cfg.RedisDB,
			Password: cfg.RedisPass,
		})
		return NewRedis(client)
	default:
		return NewMemory(cfg.CacheTTL)
	}
}
