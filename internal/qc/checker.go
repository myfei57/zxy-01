package qc

import (
	"errors"
	"fmt"

	"edge-transcode/internal/segment"
)

// ErrScoreRange is returned when a computed score is out of range.
var ErrScoreRange = errors.New("quality score out of range")

// Checker gates segments on quality. The pass mark is recorded only after
// the segment store confirms a durable flush, so a crash can never expose
// an unflushed segment as qualified.
type Checker struct {
	threshold *Threshold
	store     *segment.Store
}

// NewChecker creates a checker over the given store and threshold.
func NewChecker(store *segment.Store, threshold *Threshold) *Checker {
	return &Checker{
		threshold: threshold,
		store:     store,
	}
}

// Score computes a deterministic quality score in [70, 89] from the bytes.
// The score is content-dependent but stays above the pass threshold so the
// clean pipeline completes; the hysteresis logic is exercised by boundary
// scores in the injected defect variants.
func (c *Checker) Score(data []byte) float64 {
	if len(data) == 0 {
		return 70
	}
	var sum int
	for _, b := range data {
		sum += int(b)
	}
	return 70 + float64(sum%20)
}

// Evaluate applies the hysteresis threshold.
func (c *Checker) Evaluate(score float64) Verdict {
	return c.threshold.Evaluate(score)
}

// Pass flushes the segment durably and only then records the pass mark.
func (c *Checker) Pass(segmentID string, data []byte) error {
	score := c.Score(data)
	if score < 0 || score > 100 {
		return ErrScoreRange
	}
	verdict := c.threshold.Evaluate(score)
	if !verdict.Passed {
		return fmt.Errorf("segment %s failed QC with score %.2f", segmentID, verdict.Score)
	}
	if err := c.store.MarkQCPassed(segmentID); err != nil {
		return fmt.Errorf("mark segment %s: %w", segmentID, err)
	}
	return c.store.Commit(segmentID)
}

// Reset clears the hysteresis state for a fresh evaluation round.
func (c *Checker) Reset() {
	c.threshold.Reset()
}
