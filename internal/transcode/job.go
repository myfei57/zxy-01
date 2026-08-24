// Package transcode drives jobs through the transcode pipeline: queued,
// transcoding, qc, publishing, done or failed. State transitions are
// validated so a terminal job can never silently move backwards.
package transcode

import "fmt"

// Job states.
const (
	StateQueued      = "queued"
	StateTranscoding = "transcoding"
	StateQC          = "qc"
	StatePublishing  = "publishing"
	StateDone        = "done"
	StateFailed      = "failed"
)

// Job describes one transcode task in the pipeline.
type Job struct {
	ID        string
	StreamID  string
	SegmentID string
	Profile   string
	State     string
	Attempts  int
	Error     string
	CreatedAt string
}

// validTransitions defines the allowed state machine edges.
var validTransitions = map[string]map[string]bool{
	StateQueued:      {StateTranscoding: true, StateFailed: true},
	StateTranscoding: {StateQC: true, StateFailed: true},
	StateQC:          {StatePublishing: true, StateQueued: true, StateFailed: true},
	StatePublishing:  {StateDone: true, StateFailed: true},
}

// CanTransition reports whether the state machine allows from -> to.
func CanTransition(from, to string) bool {
	edges, ok := validTransitions[from]
	return ok && edges[to]
}

// Next applies a guarded state transition.
func (j *Job) Next(to string) error {
	if !CanTransition(j.State, to) {
		return fmt.Errorf("illegal job transition %s -> %s", j.State, to)
	}
	j.State = to
	return nil
}

// Fail moves the job to the failed state and records the error.
func (j *Job) Fail(err error) {
	j.State = StateFailed
	j.Error = err.Error()
}
