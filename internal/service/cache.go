package service

import (
	"sync"
	"time"
)

// cacheEntry holds a value alongside its expiration timestamp.
type cacheEntry[V any] struct {
	value     V
	expiresAt time.Time
}

// TTLCache is a generic, concurrency-safe cache with per-entry time-to-live.
// Expired entries are cleaned up lazily on Get.
type TTLCache[K comparable, V any] struct {
	mu      sync.RWMutex
	entries map[K]cacheEntry[V]
	ttl     time.Duration
}

// NewTTLCache creates a new TTLCache with the given default TTL for entries.
func NewTTLCache[K comparable, V any](ttl time.Duration) *TTLCache[K, V] {
	return &TTLCache[K, V]{
		entries: make(map[K]cacheEntry[V]),
		ttl:     ttl,
	}
}

// Get retrieves a value by key. If the entry exists but is expired, it is
// deleted and (zero, false) is returned.
func (c *TTLCache[K, V]) Get(key K) (V, bool) {
	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()

	if !ok {
		var zero V
		return zero, false
	}

	if time.Now().After(entry.expiresAt) {
		// Lazy cleanup: promote to write lock and delete.
		c.mu.Lock()
		// Re-check under write lock to avoid deleting a refreshed entry.
		if e, still := c.entries[key]; still && time.Now().After(e.expiresAt) {
			delete(c.entries, key)
		}
		c.mu.Unlock()

		var zero V
		return zero, false
	}

	return entry.value, true
}

// Set stores a value with the cache's default TTL.
func (c *TTLCache[K, V]) Set(key K, value V) {
	c.mu.Lock()
	c.entries[key] = cacheEntry[V]{
		value:     value,
		expiresAt: time.Now().Add(c.ttl),
	}
	c.mu.Unlock()
}

// Delete removes an entry by key, if present.
func (c *TTLCache[K, V]) Delete(key K) {
	c.mu.Lock()
	delete(c.entries, key)
	c.mu.Unlock()
}
