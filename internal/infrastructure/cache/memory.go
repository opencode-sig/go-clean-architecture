// Package cache provides pluggable cache backends (memory and Redis) that
// implement the port.Cache interface. Both are interchangeable: the factory
// NewCache selects the configured backend, so repository cache decorators do
// not change when the backend changes.
package cache

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/kun/zhisuo-server/internal/port"
)

// memoryItem is a single cached entry with its expiration deadline.
type memoryItem struct {
	value   []byte
	expires time.Time // zero time means "never expires"
}

// Memory is a concurrency-safe in-memory cache with TTL support and periodic
// janitor-style cleanup. It implements port.Cache.
type Memory struct {
	mu    sync.RWMutex
	items map[string]memoryItem
	ttl   time.Duration // default TTL when Set passes ttl <= 0; zero disables
}

// NewMemory creates a Memory cache with the given default TTL.
func NewMemory(defaultTTL time.Duration) *Memory {
	return &Memory{
		items: make(map[string]memoryItem),
		ttl:   defaultTTL,
	}
}

// Get implements port.Cache. It returns port.ErrCacheMiss when the key is
// missing or already expired.
func (m *Memory) Get(_ context.Context, key string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	item, ok := m.items[key]
	if !ok {
		return nil, port.ErrCacheMiss
	}

	if !item.expires.IsZero() && time.Now().After(item.expires) {
		delete(m.items, key)
		return nil, port.ErrCacheMiss
	}

	return item.value, nil
}

// Set implements port.Cache.
func (m *Memory) Set(_ context.Context, key string, value []byte, ttl time.Duration) error {
	if ttl <= 0 {
		ttl = m.ttl
	}
	if ttl <= 0 {
		ttl = 0 // no expiration
	}

	var expires time.Time
	if ttl > 0 {
		expires = time.Now().Add(ttl)
	}

	m.mu.Lock()
	m.items[key] = memoryItem{value: value, expires: expires}
	m.mu.Unlock()

	return nil
}

// Del implements port.Cache. Deleting a non-existent key is a no-op.
func (m *Memory) Del(_ context.Context, keys ...string) error {
	m.mu.Lock()
	for _, key := range keys {
		delete(m.items, key)
	}
	m.mu.Unlock()

	return nil
}

// DeleteByPrefix removes all keys starting with the given prefix.
func (m *Memory) DeleteByPrefix(_ context.Context, prefix string) error {
	m.mu.Lock()
	for key := range m.items {
		if strings.HasPrefix(key, prefix) {
			delete(m.items, key)
		}
	}
	m.mu.Unlock()

	return nil
}

// compile-time assertions: Memory implements port.Cache and CacheListInvalidator
var _ port.Cache = (*Memory)(nil)
var _ port.CacheListInvalidator = (*Memory)(nil)
