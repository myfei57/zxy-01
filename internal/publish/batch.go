package publish

import (
	"fmt"

	"edge-transcode/internal/segment"
)

// Batch pushes a set of segments to the CDN. The cursor advances only when
// every segment of the batch succeeded, so a partial failure never marks
// missing content as published.
type Batch struct {
	cdn    *CDN
	cursor *Cursor
	store  *segment.Store
}

// NewBatch creates a publish batch.
func NewBatch(cdn *CDN, cursor *Cursor, store *segment.Store) *Batch {
	return &Batch{cdn: cdn, cursor: cursor, store: store}
}

// Push publishes the given segment ids in order. On the first failure it
// returns an error and leaves the cursor unchanged.
func (b *Batch) Push(streamID string, segmentIDs []string) error {
	for _, id := range segmentIDs {
		data, err := b.store.Get(id)
		if err != nil {
			return fmt.Errorf("publish %s: read segment %s: %w", streamID, id, err)
		}
		if err := b.cdn.Push(id, data); err != nil {
			return fmt.Errorf("publish %s: cdn push %s: %w", streamID, id, err)
		}
	}
	b.cursor.Advance(streamID, len(segmentIDs))
	return nil
}
