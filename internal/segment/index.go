// Package segment implements the media storage mechanics of EdgeTranscode:
// index visibility, durable writes, committed segments and chunk
// reassembly. It has no knowledge of streams, jobs or QC.
package segment

import (
	"sort"
	"sync"
)

// IndexEntry is the visible metadata of one stored segment.
type IndexEntry struct {
	ID   string `json:"id"`
	Seq  int    `json:"seq"`
	Size int64  `json:"size"`
	Path string `json:"path"`
}

// Index maps segment ids to their visible entries.
type Index struct {
	mu      sync.RWMutex
	entries map[string]IndexEntry
}

// NewIndex creates an empty index.
func NewIndex() *Index {
	return &Index{entries: make(map[string]IndexEntry)}
}

// Update records or replaces the visible entry for a segment. Callers must
// only publish an entry after the underlying bytes are fully written.
func (ix *Index) Update(entry IndexEntry) {
	ix.mu.Lock()
	defer ix.mu.Unlock()
	ix.entries[entry.ID] = entry
}

// Resolve returns the visible entry of a segment.
func (ix *Index) Resolve(id string) (IndexEntry, bool) {
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	entry, ok := ix.entries[id]
	return entry, ok
}

// List returns all entries sorted by sequence number.
func (ix *Index) List() []IndexEntry {
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	out := make([]IndexEntry, 0, len(ix.entries))
	for _, entry := range ix.entries {
		out = append(out, entry)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Seq < out[j].Seq })
	return out
}
