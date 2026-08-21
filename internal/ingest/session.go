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

// Cleanup terminates a session. It releases the in-flight write lock,
// discards any buffered in-flight chunks and removes the session from the
// registry so a later retry of the same id can never block on a stale lock or
// inherit half-uploaded data. A session must always end here, whether it
// completes or fails; it must never stop half-way.
func (m *Manager) Cleanup(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	session, ok := m.sessions[id]
	if !ok {
		return
	}
	// Drop the write lock first so nothing can observe a half-released
	// session, then discard the in-flight chunk buffer.
	session.ReleaseLock()
	if session.reassembly != nil {
		session.reassembly.Reset()
	}
	delete(m.sessions, id)
}
