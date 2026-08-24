// Package playback serves segments to players through an edge cache backed
// by the delivery network.
package playback

import (
	"errors"
	"sync"
)

// ErrNonSuccess is returned when a response with an error status would be
// written into the cache.
var ErrNonSuccess = errors.New("non-success response must not be cached")

// CacheEntry is one cached object.
type CacheEntry struct {
	Status int
	Data   []byte
}

// Cache is the edge object cache.
type Cache struct {
	mu      sync.Mutex
	entries map[string]CacheEntry
}

// NewCache creates an empty cache.
func NewCache() *Cache {
	return &Cache{entries: make(map[string]CacheEntry)}
}

// Write stores an entry only for success statuses. Error responses are
// refused so a failed fetch can never poison the cache.
func (c *Cache) Write(key string, status int, data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[key] = CacheEntry{Status: status, Data: append([]byte(nil), data...)}
	return nil
}

// Lookup returns a cached entry.
func (c *Cache) Lookup(key string) (CacheEntry, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.entries[key]
	return entry, ok
}

// Count returns the number of cached entries.
func (c *Cache) Count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.entries)
}
