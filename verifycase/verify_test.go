package verifycase

import (
	"testing"

	"edge-transcode/internal/publish"
	"edge-transcode/internal/segment"
)

// TestPublishBatchCursorHoldsOnPartialFailure verifies the cursor never
// advances when any segment of the batch fails.
func TestPublishBatchCursorHoldsOnPartialFailure(t *testing.T) {
	st := segment.NewStore(t.TempDir())
	cdn := publish.NewCDN()
	cursor := publish.NewCursor()
	batch := publish.NewBatch(cdn, cursor, st)
	if err := batch.Push("stream-1", []string{"missing-seg"}); err == nil {
		t.Fatal("batch must fail when a segment is missing")
	}
	if got := cursor.Position("stream-1"); got != 0 {
		t.Fatalf("cursor advanced on failure: %d", got)
	}
}
