package store

import (
	"path/filepath"
	"sort"
	"sync"
)

// State is the central in-memory + file-backed registry of the service.
type State struct {
	mu        sync.RWMutex
	root      string
	streams   map[string]*Stream
	segments  map[string]*Segment
	jobs      map[string]*Job
	manifests map[string]*Manifest
	cursors   map[string]*PublishCursor
	samples   []QualitySample
}

// NewState creates a State rooted at dir.
func NewState(dir string) *State {
	return &State{
		root:      dir,
		streams:   make(map[string]*Stream),
		segments:  make(map[string]*Segment),
		jobs:      make(map[string]*Job),
		manifests: make(map[string]*Manifest),
		cursors:   make(map[string]*PublishCursor),
	}
}

// CreateStream registers a new stream.
func (s *State) CreateStream(stream *Stream) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.streams[stream.ID] = stream
}

// Stream returns a stream by id.
func (s *State) Stream(id string) (*Stream, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	stream, ok := s.streams[id]
	return stream, ok
}

// UpdateStream replaces the stored stream value.
func (s *State) UpdateStream(stream *Stream) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.streams[stream.ID] = stream
}

// ListStreams returns streams sorted by id.
func (s *State) ListStreams() []*Stream {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := make([]string, 0, len(s.streams))
	for id := range s.streams {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]*Stream, 0, len(ids))
	for _, id := range ids {
		out = append(out, s.streams[id])
	}
	return out
}

// PutSegment stores a segment record.
func (s *State) PutSegment(segment *Segment) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.segments[segment.ID] = segment
}

// ListSegments returns all segments sorted by stream then sequence.
func (s *State) ListSegments() []*Segment {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Segment, 0, len(s.segments))
	for _, segment := range s.segments {
		out = append(out, segment)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].StreamID != out[j].StreamID {
			return out[i].StreamID < out[j].StreamID
		}
		return out[i].Seq < out[j].Seq
	})
	return out
}

// PutJob stores a job record.
func (s *State) PutJob(job *Job) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs[job.ID] = job
}

// Job returns a job by id.
func (s *State) Job(id string) (*Job, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	job, ok := s.jobs[id]
	return job, ok
}

// ListJobs returns jobs sorted by creation order (oldest first).
func (s *State) ListJobs() []*Job {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := make([]string, 0, len(s.jobs))
	for id := range s.jobs {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]*Job, 0, len(ids))
	for _, id := range ids {
		out = append(out, s.jobs[id])
	}
	return out
}

// PutManifest stores a manifest revision.
func (s *State) PutManifest(manifest *Manifest) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.manifests[manifest.StreamID] = manifest
	return SaveJSON(s.manifestPath(manifest.StreamID), manifest)
}

// Manifest returns the latest manifest for a stream.
func (s *State) Manifest(streamID string) (*Manifest, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	manifest, ok := s.manifests[streamID]
	return manifest, ok
}

// SetCursor records the published position of a stream.
func (s *State) SetCursor(streamID string, position int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cursors[streamID] = &PublishCursor{StreamID: streamID, Position: position}
}

// Cursor returns the published position of a stream.
func (s *State) Cursor(streamID string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cursor, ok := s.cursors[streamID]
	if !ok {
		return 0
	}
	return cursor.Position
}

// AddSample appends a QC sample.
func (s *State) AddSample(sample QualitySample) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.samples = append(s.samples, sample)
}

// Samples returns recent samples, newest last.
func (s *State) Samples() []QualitySample {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]QualitySample, len(s.samples))
	copy(out, s.samples)
	return out
}

// Save persists streams, segments, jobs and cursors to disk.
func (s *State) Save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	payload := map[string]any{
		"streams":  s.streams,
		"segments": s.segments,
		"jobs":     s.jobs,
		"cursors":  s.cursors,
	}
	return SaveJSON(filepath.Join(s.root, "state.json"), payload)
}

// Load restores previously saved state from disk.
func (s *State) Load() error {
	payload := struct {
		Streams  map[string]*Stream        `json:"streams"`
		Segments map[string]*Segment       `json:"segments"`
		Jobs     map[string]*Job           `json:"jobs"`
		Cursors  map[string]*PublishCursor `json:"cursors"`
	}{}
	if err := LoadJSON(filepath.Join(s.root, "state.json"), &payload); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, stream := range payload.Streams {
		s.streams[id] = stream
	}
	for id, segment := range payload.Segments {
		s.segments[id] = segment
	}
	for id, job := range payload.Jobs {
		s.jobs[id] = job
	}
	for id, cursor := range payload.Cursors {
		s.cursors[id] = cursor
	}
	return nil
}

func (s *State) manifestPath(streamID string) string {
	return filepath.Join(s.root, "manifests", streamID+".json")
}
