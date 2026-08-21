package manifest

import (
	"fmt"

	"edge-transcode/internal/segment"
)

// Builder assembles manifests from the segment store. A manifest version
// only advances when the referenced sequence is contiguous, so playback can
// never reference a missing segment.
type Builder struct {
	store *segment.Store
}

// NewBuilder creates a manifest builder over the segment store.
func NewBuilder(store *segment.Store) *Builder {
	return &Builder{store: store}
}

// Build returns the ordered segment ids for a generation, or an error when
// the sequence has a gap or the generation is inconsistent. The manifest
// must not advance past a gap, so the error is surfaced instead of being
// swallowed: a missing segment stops publication in place.
func (b *Builder) Build(streamID string, generation int64, max int) ([]string, error) {
	ids, err := b.store.OrderedSegments(generation, max)
	if err != nil {
		return nil, fmt.Errorf("build manifest %s generation %d: %w", streamID, generation, err)
	}
	return ids, nil
}
