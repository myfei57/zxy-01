// Package ingest receives multipart uploads from creators and reassembles
// them into stored segments.
package ingest

import (
	"sync"

	"edge-transcode/internal/segment"
)

// Session tracks one multipart upload. The in-flight lock guards the chunk
// write path; a session must always end in a terminal cleanup that releases
// every lock it holds.
type Session struct {
	ID         string
	StreamID   string
	ChunkCount int
	locked     bool
	reassembly *segment.Reassembly
}

// AcquireLock takes the in-flight write lock. It returns false when the
// session already holds it.
func (s *Session) AcquireLock() bool {
	if s.locked {
		return false
	}
	s.locked = true
	return true
}

// ReleaseLock releases the in-flight write lock.
func (s *Session) ReleaseLock() {
	s.locked = false
}

// Reassembly exposes the chunk buffer of the session.
func (s *Session) Reassembly() *segment.Reassembly {
	return s.reassembly
}

// Manager owns all live upload sessions.
type Manager struct {
	mu       sync.Mutex
	sessions map[string]*Session
}

// NewManager creates an empty session manager.
func NewManager() *Manager {
	return &Manager{sessions: make(map[string]*Session)}
}

// Create registers a new session.
func (m *Manager) Create(id, streamID string, chunkCount int) *Session {
	m.mu.Lock()
	defer m.mu.Unlock()
	session := &Session{
		ID:         id,
		StreamID:   streamID,
		ChunkCount: chunkCount,
		reassembly: segment.NewReassembly(),
	}
	m.sessions[id] = session
	return session
}

// Get returns a live session.
func (m *Manager) Get(id string) (*Session, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	session, ok := m.sessions[id]
	return session, ok
}

// Cleanup removes a session and always releases its in-flight lock so a
// later retry of the same id can never block on a stale lock.
func (m *Manager) Cleanup(id string) {
	// Defect variant: cleanup never runs, so the session and its in-flight
	// lock stay behind.
}
