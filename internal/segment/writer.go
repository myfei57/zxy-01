package segment

import (
	"fmt"
	"os"
	"path/filepath"
)

// Writer persists segment bytes and then publishes the index entry, so a
// reader can only ever resolve a fully written segment.
type Writer struct {
	root  string
	index *Index
}

// NewWriter creates a Writer rooted at dir backed by index.
func NewWriter(dir string, index *Index) *Writer {
	return &Writer{root: dir, index: index}
}

// Write durably stores data and updates the index only after the bytes are
// on disk. The returned entry is visible to readers afterwards.
func (w *Writer) Write(id string, seq int, data []byte) (IndexEntry, error) {
	if err := os.MkdirAll(w.root, 0o755); err != nil {
		return IndexEntry{}, fmt.Errorf("segment dir: %w", err)
	}
	path := filepath.Join(w.root, id+".seg")
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return IndexEntry{}, fmt.Errorf("write segment %s: %w", id, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return IndexEntry{}, fmt.Errorf("commit segment %s: %w", id, err)
	}
	entry := IndexEntry{ID: id, Seq: seq, Size: int64(len(data)), Path: path}
	w.index.Update(entry)
	return entry, nil
}
