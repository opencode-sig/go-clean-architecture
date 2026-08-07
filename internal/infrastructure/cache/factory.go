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
		return NewRedis(newUniversalClient(cfg))
	default:
		return NewMemory(cfg.CacheTTL)
	}
}

// newUniversalClient builds a redis client for either topology.
// cfg.RedisMode selects the topology: "cluster" uses cfg.RedisAddrs as the
// seed nodes (cluster requires them), anything else falls back to the single
// node host:port.
func newUniversalClient(cfg *infrastructure.Config) redis.UniversalClient {
	if cfg.RedisMode == "cluster" {
		return redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:    cfg.RedisAddrs,
			Password: cfg.RedisPass,
		})
	}

	return redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.RedisHost, cfg.RedisPort),
		DB:       cfg.RedisDB,
		Password: cfg.RedisPass,
	})
}
