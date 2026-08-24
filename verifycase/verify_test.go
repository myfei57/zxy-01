package verifycase

import (
	"bytes"
	"testing"

	"edge-transcode/internal/segment"
)

// TestChunkReassemblyUsesSequenceOrder verifies the reassembled file follows
// the chunk sequence, not the arrival order.
func TestChunkReassemblyUsesSequenceOrder(t *testing.T) {
	r := segment.NewReassembly()
	if err := r.Add(2, []byte("B")); err != nil {
		t.Fatal(err)
	}
	if err := r.Add(1, []byte("A")); err != nil {
		t.Fatal(err)
	}
	got, err := r.Finalize(2)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, []byte("AB")) {
		t.Fatalf("reassembled order mismatch: got %q want %q", got, "AB")
	}
}
