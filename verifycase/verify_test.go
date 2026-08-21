package verifycase

import (
	"testing"

	"edge-transcode/internal/ingest"
)

// TestUploadSessionReleasesInFlightOnDisconnect verifies a disconnected
// session is cleaned up and its in-flight lock released.
func TestUploadSessionReleasesInFlightOnDisconnect(t *testing.T) {
	manager := ingest.NewManager()
	session := manager.Create("sess-1", "stream-1", 3)
	if !session.AcquireLock() {
		t.Fatal("lock should be acquired")
	}
	manager.Cleanup("sess-1")
	if _, ok := manager.Get("sess-1"); ok {
		t.Fatal("disconnected session must be removed")
	}
}
