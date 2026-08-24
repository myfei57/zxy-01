package manifest

import (
	"fmt"

	"edge-transcode/internal/store"
)

// Publisher publishes manifest revisions. The revision is persisted before
// the checkpoint is written, so recovery never treats an unpublished
// revision as published after a crash.
type Publisher struct {
	state      *store.State
	checkpoint *store.Checkpoint
}

// NewPublisher creates a manifest publisher.
func NewPublisher(state *store.State, checkpoint *store.Checkpoint) *Publisher {
	return &Publisher{state: state, checkpoint: checkpoint}
}

// Publish persists the manifest revision durably and then advances the
// checkpoint to that revision.
func (p *Publisher) Publish(m *Manifest) error {
	m.Published = true
	record := &store.Manifest{
		StreamID:   m.StreamID,
		Revision:   m.Revision,
		Generation: m.Generation,
		SegmentIDs: m.SegmentIDs,
		Published:  m.Published,
	}
	if err := p.state.PutManifest(record); err != nil {
		return fmt.Errorf("persist manifest %s rev %d: %w", m.StreamID, m.Revision, err)
	}
	return p.checkpoint.Write(m.StreamID, m.Revision)
}

// RecoveryRevision reports the checkpoint revision of a stream, used after
// a restart to know which revision is durably published.
func (p *Publisher) RecoveryRevision(streamID string) (int64, error) {
	return p.checkpoint.Read(streamID)
}
