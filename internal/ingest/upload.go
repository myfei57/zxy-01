package ingest

import (
	"errors"
	"fmt"

	"edge-transcode/internal/segment"
)

// ErrNotLocked is returned when a chunk arrives for a session whose
// in-flight lock is not held.
var ErrNotLocked = errors.New("upload session not locked")

// Service receives chunks and completes uploads into the segment store.
type Service struct {
	manager *Manager
	store   *segment.Store
}

// NewService creates an upload service.
func NewService(manager *Manager, store *segment.Store) *Service {
	return &Service{manager: manager, store: store}
}

// Receive buffers one chunk for the session under its in-flight lock.
func (s *Service) Receive(sessionID string, seq int, data []byte) error {
	session, ok := s.manager.Get(sessionID)
	if !ok {
		return fmt.Errorf("unknown upload session %s", sessionID)
	}
	if !session.AcquireLock() {
		return ErrNotLocked
	}
	defer session.ReleaseLock()
	return session.Reassembly().Add(seq, data)
}

// Complete reassembles the upload, persists the resulting segment and
// terminates the session. The returned id is the stored segment id.
func (s *Service) Complete(sessionID, streamID string, generation int64) (string, error) {
	session, ok := s.manager.Get(sessionID)
	if !ok {
		return "", fmt.Errorf("unknown upload session %s", sessionID)
	}
	if !session.AcquireLock() {
		return "", ErrNotLocked
	}
	defer session.ReleaseLock()
	data, err := session.Reassembly().Finalize(session.ChunkCount)
	if err != nil {
		s.manager.Cleanup(sessionID)
		return "", fmt.Errorf("reassemble: %w", err)
	}
	// The reassembled upload becomes one source segment; transcode later
	// splits it into numbered output segments.
	segmentID := streamID + "-src"
	if _, err := s.store.Put(segmentID, 0, generation, data); err != nil {
		s.manager.Cleanup(sessionID)
		return "", fmt.Errorf("store segment: %w", err)
	}
	s.manager.Cleanup(sessionID)
	return segmentID, nil
}

// Abort terminates a session without persisting anything.
func (s *Service) Abort(sessionID string) {
	s.manager.Cleanup(sessionID)
}
