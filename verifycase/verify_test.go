package verifycase

import (
	"testing"

	"edge-transcode/internal/qc"
)

// TestQcThresholdHysteresisStable verifies a boundary score cannot flip an
// established verdict.
func TestQcThresholdHysteresisStable(t *testing.T) {
	th := qc.NewThreshold(45, 65)
	if !th.Evaluate(70).Passed {
		t.Fatal("score above high must pass")
	}
	if !th.Evaluate(50).Passed {
		t.Fatal("hysteresis band must keep the passed verdict at boundary scores")
	}
}
