// Package qc implements the quality gate: deterministic scoring, a
// hysteresis threshold and durable pass marking after segment flush.
package qc

import "sync"

// Verdict is the outcome of one quality evaluation.
type Verdict struct {
	Passed bool
	Score  float64
}

// Threshold evaluates scores with a hysteresis band. Once a segment is
// judged passed, it stays passed until the score drops below Low; once
// failed, it stays failed until the score rises above High. A score inside
// the band never flips the verdict, which keeps boundary samples stable.
type Threshold struct {
	mu    sync.Mutex
	low   float64
	high  float64
	state bool
}

// NewThreshold creates a threshold with the given band.
func NewThreshold(low, high float64) *Threshold {
	if low >= high {
		low, high = high, low
	}
	return &Threshold{low: low, high: high}
}

// Evaluate applies the hysteresis band and returns the verdict.
func (t *Threshold) Evaluate(score float64) Verdict {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.state {
		if score < t.low {
			t.state = false
		}
	} else {
		if score > t.high {
			t.state = true
		}
	}
	return Verdict{Passed: t.state, Score: score}
}

// Reset clears the remembered verdict for a fresh segment.
func (t *Threshold) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.state = false
}
