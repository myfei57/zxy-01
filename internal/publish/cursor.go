package publish

import "sync"

// Cursor tracks how many segments of a stream reached the CDN.
type Cursor struct {
	mu       sync.Mutex
	position map[string]int
}

// NewCursor creates an empty cursor book.
func NewCursor() *Cursor {
	return &Cursor{position: make(map[string]int)}
}

// Position returns the published position of a stream.
func (c *Cursor) Position(streamID string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.position[streamID]
}

// Advance moves the cursor only forward.
func (c *Cursor) Advance(streamID string, to int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if to > c.position[streamID] {
		c.position[streamID] = to
	}
}
