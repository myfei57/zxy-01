// Package publish delivers transcoded segments to the delivery network and
// tracks the published cursor of every stream.
package publish

import (
	"errors"
	"sync"
)

// ErrObjectNotFound is returned when the CDN has no object.
var ErrObjectNotFound = errors.New("cdn object not found")

// CDN is the in-process delivery network.
type CDN struct {
	mu      sync.Mutex
	objects map[string][]byte
}

// NewCDN creates an empty CDN.
func NewCDN() *CDN {
	return &CDN{objects: make(map[string][]byte)}
}

// Push stores one object.
func (c *CDN) Push(id string, data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.objects[id] = append([]byte(nil), data...)
	return nil
}

// Get returns a stored object.
func (c *CDN) Get(id string) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	data, ok := c.objects[id]
	if !ok {
		return nil, ErrObjectNotFound
	}
	return append([]byte(nil), data...), nil
}

// Count returns the number of objects.
func (c *CDN) Count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.objects)
}
