package segment

import (
	"errors"
	"fmt"
	"os"
	"sync"
)

// ErrNotFound is returned when a segment is unknown.
var ErrNotFound = errors.New("segment not found")

// Meta describes a committed segment.
type Meta struct {
	ID         string
	Seq        int
	Size       int64
	Generation int64
	State      string
}

// Store manages committed segments and their lifecycle states.
type Store struct {
	mu         sync.RWMutex
	writer     *Writer
	index      *Index
	committed  map[string]bool
	states     map[string]string
	generation map[string]int64
}

// NewStore creates a Store over dir with a fresh index.
func NewStore(dir string) *Store {
	index := NewIndex()
	return &Store{
		writer:     NewWriter(dir, index),
		index:      index,
		committed:  make(map[string]bool),
		states:     make(map[string]string),
		generation: make(map[string]int64),
	}
}

// Put writes a new segment and marks it committed immediately.
func (s *Store) Put(id string, seq int, generation int64, data []byte) (Meta, error) {
	entry, err := s.writer.Write(id, seq, data)
	if err != nil {
		return Meta{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.states[id] = "stored"
	s.generation[id] = generation
	return Meta{
		ID:         id,
		Seq:        entry.Seq,
		Size:       entry.Size,
		Generation: generation,
		State:      "stored",
	}, nil
}

// Commit marks a segment durably committed; this is the flush confirmation
// that QC waits for before recording its pass mark.
func (s *Store) Commit(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.index.Resolve(id); !ok {
		return ErrNotFound
	}
	s.committed[id] = true
	return nil
}

// MarkQCPassed advances a committed segment to the QC-passed state.
func (s *Store) MarkQCPassed(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.committed[id] {
		return fmt.Errorf("segment %s not committed", id)
	}
	s.states[id] = "qc_passed"
	return nil
}

// Get reads the stored bytes of a segment.
func (s *Store) Get(id string) ([]byte, error) {
	entry, ok := s.index.Resolve(id)
	if !ok {
		return nil, ErrNotFound
	}
	data, err := os.ReadFile(entry.Path)
	if err != nil {
		return nil, fmt.Errorf("read segment %s: %w", id, err)
	}
	return data, nil
}

// Meta returns the metadata of a segment.
func (s *Store) Meta(id string) (Meta, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, ok := s.index.Resolve(id)
	if !ok {
		return Meta{}, false
	}
	return Meta{
		ID:         entry.ID,
		Seq:        entry.Seq,
		Size:       entry.Size,
		Generation: s.generation[id],
		State:      s.states[id],
	}, true
}

// Index exposes the underlying index for readers.
func (s *Store) Index() *Index {
	return s.index
}

// OrderedSegments validates that every sequence 1..max exists in generation
// and returns the segment ids in sequence order. A missing sequence is an
// error, so a manifest can never be built over a gap.
func (s *Store) OrderedSegments(generation int64, max int) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	bySeq := make(map[int]string)
	for id, gen := range s.generation {
		if gen != generation {
			continue
		}
		entry, ok := s.index.Resolve(id)
		if !ok {
			continue
		}
		bySeq[entry.Seq] = id
	}
	out := make([]string, 0, max)
	for seq := 1; seq <= max; seq++ {
		id, ok := bySeq[seq]
		if !ok {
			return nil, fmt.Errorf("missing segment sequence %d in generation %d", seq, generation)
		}
		out = append(out, id)
	}
	return out, nil
}
