package verifycase

import (
	"os"
	"path/filepath"
	"testing"

	"edge-transcode/internal/segment"
)

// TestSegmentIndexNeverShowsHalfWrittenSegment verifies the index entry is
// only visible after the segment bytes are fully written.
func TestSegmentIndexNeverShowsHalfWrittenSegment(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	index := segment.NewIndex()
	writer := segment.NewWriter(blocker, index)
	if _, err := writer.Write("seg1", 1, []byte("data")); err == nil {
		t.Fatal("write to invalid root must fail")
	}
	if _, ok := index.Resolve("seg1"); ok {
		t.Fatal("index must not expose a segment whose bytes were not written")
	}
}
