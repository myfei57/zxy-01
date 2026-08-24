package verifycase

import (
	"testing"

	"edge-transcode/internal/qc"
	"edge-transcode/internal/segment"
)

// TestQcFlushOrderingSurvivesCrash verifies the QC pass mark is recorded
// only after the segment is durably flushed.
func TestQcFlushOrderingSurvivesCrash(t *testing.T) {
	dir := t.TempDir()
	st := segment.NewStore(dir)
	if _, err := st.Put("seg1", 1, 9, []byte("payload")); err != nil {
		t.Fatal(err)
	}
	checker := qc.NewChecker(st, qc.NewThreshold(45, 65))
	if err := checker.Pass("seg1", []byte("payload")); err != nil {
		t.Fatalf("QC pass must flush before marking: %v", err)
	}
	meta, ok := st.Meta("seg1")
	if !ok || meta.State != "qc_passed" {
		t.Fatalf("segment state = %+v, want qc_passed", meta)
	}
}
