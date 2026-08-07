package cache

import (
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

// newUniversalClient builds a redis client for either topology. Both modes
// read addresses from cfg.RedisAddrs (uniform) and authenticate with
// cfg.RedisUser/cfg.RedisPass (ACL): "cluster" feeds the addrs to go-redis as
// seed nodes; "single" uses the first entry and applies cfg.RedisDB.
func newUniversalClient(cfg *infrastructure.Config) redis.UniversalClient {
	if cfg.RedisMode == "cluster" {
		return redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:    cfg.RedisAddrs,
			Username: cfg.RedisUser,
			Password: cfg.RedisPass,
		})
	}

	return redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddrs[0],
		DB:       cfg.RedisDB,
		Username: cfg.RedisUser,
		Password: cfg.RedisPass,
	})
}
