package verifycase

import (
	"os"
	"path/filepath"
	"testing"

	"edge-transcode/internal/manifest"
	"edge-transcode/internal/store"
)

// TestManifestCheckpointOrderAfterCrash verifies the checkpoint advances
// only after the manifest revision is durably persisted.
func TestManifestCheckpointOrderAfterCrash(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	state := store.NewState(blocker)
	checkpoint := store.NewCheckpoint(t.TempDir())
	publisher := manifest.NewPublisher(state, checkpoint)
	if err := publisher.Publish(manifest.New("stream-1", 1, 1, []string{"s1"})); err == nil {
		t.Fatal("publish must fail when the manifest revision cannot be persisted")
	}
	rev, err := checkpoint.Read("stream-1")
	if err != nil {
		t.Fatal(err)
	}
	if rev != 0 {
		t.Fatalf("checkpoint advanced to %d although the revision was never persisted", rev)
	}
}
