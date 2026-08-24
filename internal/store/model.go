// Package store owns the durable and in-memory state of the EdgeTranscode
// service: streams, segments, jobs, manifests, cursors and quality samples.
package store

// Stream lifecycle states.
const (
	StreamDraft   = "draft"
	StreamStaging = "staging"
	StreamLive    = "live"
	StreamFailed  = "failed"
)

// Segment pipeline states.
const (
	SegmentStored    = "stored"
	SegmentQCPassed  = "qc_passed"
	SegmentPublished = "published"
)

// Stream describes one uploaded video and its lifecycle.
type Stream struct {
	ID         string    `json:"id"`
	Title      string    `json:"title"`
	Owner      string    `json:"owner"`
	State      string    `json:"state"`
	Generation int64     `json:"generation"`
	Segments   []Segment `json:"segments"`
	CreatedAt  string    `json:"created_at"`
}

// Segment describes one media segment of a stream.
type Segment struct {
	ID         string `json:"id"`
	StreamID   string `json:"stream_id"`
	Seq        int    `json:"seq"`
	Size       int64  `json:"size"`
	Path       string `json:"path"`
	State      string `json:"state"`
	Generation int64  `json:"generation"`
	Digest     string `json:"digest"`
}

// Job describes one transcode task.
type Job struct {
	ID        string `json:"id"`
	StreamID  string `json:"stream_id"`
	SegmentID string `json:"segment_id"`
	Profile   string `json:"profile"`
	State     string `json:"state"`
	Attempts  int    `json:"attempts"`
	Error     string `json:"error,omitempty"`
	CreatedAt string `json:"created_at"`
}

// Manifest describes a published playlist revision.
type Manifest struct {
	StreamID   string   `json:"stream_id"`
	Revision   int64    `json:"revision"`
	Generation int64    `json:"generation"`
	SegmentIDs []string `json:"segment_ids"`
	Published  bool     `json:"published"`
}

// PublishCursor records how many segments of a stream reached the CDN.
type PublishCursor struct {
	StreamID string `json:"stream_id"`
	Position int    `json:"position"`
}

// QualitySample records one QC outcome.
type QualitySample struct {
	StreamID  string  `json:"stream_id"`
	SegmentID string  `json:"segment_id"`
	Score     float64 `json:"score"`
	Passed    bool    `json:"passed"`
	At        string  `json:"at"`
}
