package verifycase

import (
	"os"
	"path/filepath"
	"testing"

	"edge-transcode/internal/segment"
	"edge-transcode/internal/transcode"
)

// TestTranscodeOutputWriteErrorFailsJob verifies a segment store write
// failure fails the transcode instead of being swallowed.
func TestTranscodeOutputWriteErrorFailsJob(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	st := segment.NewStore(blocker)
	worker := transcode.NewWorker(st)
	if _, err := worker.Transcode("stream-1", 1, []byte("source")); err == nil {
		t.Fatal("transcode must fail when output write fails")
	}
}
