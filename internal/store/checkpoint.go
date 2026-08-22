package store

import (
	"fmt"
	"path/filepath"
)

// Checkpoint durably records which manifest revision of a stream is
// considered published. Recovery reads it after a restart.
type Checkpoint struct {
	root string
}

// checkpointRecord is the persisted shape.
type checkpointRecord struct {
	StreamID string `json:"stream_id"`
	Revision int64  `json:"revision"`
}

// NewCheckpoint creates a Checkpoint rooted at dir.
func NewCheckpoint(dir string) *Checkpoint {
	return &Checkpoint{root: dir}
}

// Write durably records the given revision for a stream.
func (c *Checkpoint) Write(streamID string, revision int64) error {
	record := checkpointRecord{StreamID: streamID, Revision: revision}
	return SaveJSON(c.path(streamID), record)
}

// Read returns the recorded revision for a stream; 0 when none exists.
func (c *Checkpoint) Read(streamID string) (int64, error) {
	var record checkpointRecord
	if err := LoadJSON(c.path(streamID), &record); err != nil {
		return 0, fmt.Errorf("read checkpoint %s: %w", streamID, err)
	}
	return record.Revision, nil
}

func (c *Checkpoint) path(streamID string) string {
	return filepath.Join(c.root, "checkpoints", streamID+".json")
}
