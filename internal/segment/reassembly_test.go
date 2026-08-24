package segment

import (
	"bytes"
	"testing"
)

// TestReassemblyOutOfOrder verifies that chunks arriving out of order are
// reassembled strictly by sequence number, not by arrival order. This is the
// regression for the bug where retransmits caused the server to concatenate
// chunks in arrival order and produce a file whose md5 did not match.
func TestReassemblyOutOfOrder(t *testing.T) {
	chunks := map[int][]byte{
		1: []byte("AAAA"),
		2: []byte("BBBB"),
		3: []byte("CCCC"),
		4: []byte("DDDD"),
	}
	want := bytes.Join([][]byte{chunks[1], chunks[2], chunks[3], chunks[4]}, nil)

	// Arrive in reverse order, as a retransmit storm might deliver them.
	r := NewReassembly()
	for seq := 4; seq >= 1; seq-- {
		if err := r.Add(seq, chunks[seq]); err != nil {
			t.Fatalf("Add(%d): %v", seq, err)
		}
	}
	got, err := r.Finalize(4)
	if err != nil {
		t.Fatalf("Finalize: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("reassembled bytes wrong\ngot  %q\nwant %q", got, want)
	}
}

// TestReassemblyGapStalls verifies that a missing chunk stalls the output:
// the bytes of later chunks are not emitted until the gap is filled.
func TestReassemblyGapStalls(t *testing.T) {
	r := NewReassembly()
	// Chunk 2 arrives first; since chunk 1 is missing, nothing is drained.
	if err := r.Add(2, []byte("BBBB")); err != nil {
		t.Fatalf("Add(2): %v", err)
	}
	if got := r.pending(); len(got) != 0 {
		t.Fatalf("expected no drained bytes while chunk 1 missing, got %d", len(got))
	}
	// Chunk 1 fills the gap; now the contiguous prefix 1,2 drains in order.
	if err := r.Add(1, []byte("AAAA")); err != nil {
		t.Fatalf("Add(1): %v", err)
	}
	if err := r.Add(3, []byte("CCCC")); err != nil {
		t.Fatalf("Add(3): %v", err)
	}
	got, err := r.Finalize(3)
	if err != nil {
		t.Fatalf("Finalize: %v", err)
	}
	want := []byte("AAAABBBBCCCC")
	if !bytes.Equal(got, want) {
		t.Fatalf("reassembled bytes wrong\ngot  %q\nwant %q", got, want)
	}
}

// TestReassemblyDuplicateRejected verifies that a duplicate sequence is
// rejected rather than silently appended twice.
func TestReassemblyDuplicateRejected(t *testing.T) {
	r := NewReassembly()
	if err := r.Add(1, []byte("AAAA")); err != nil {
		t.Fatalf("Add(1): %v", err)
	}
	err := r.Add(1, []byte("XXXX"))
	if err == nil {
		t.Fatal("expected error for duplicate chunk, got nil")
	}
}

// TestReassemblyMissingChunkFinalize verifies that Finalize fails when an
// expected sequence never arrived, so a corrupt/incomplete file is never
// produced.
func TestReassemblyMissingChunkFinalize(t *testing.T) {
	r := NewReassembly()
	_ = r.Add(1, []byte("AAAA"))
	_ = r.Add(3, []byte("CCCC")) // chunk 2 never arrived
	if _, err := r.Finalize(3); err == nil {
		t.Fatal("expected Finalize to fail on missing chunk 2")
	}
}

// TestReassemblyFinalizeTwice verifies that a finalized reassembly cannot be
// finalized again.
func TestReassemblyFinalizeTwice(t *testing.T) {
	r := NewReassembly()
	_ = r.Add(1, []byte("AAAA"))
	if _, err := r.Finalize(1); err != nil {
		t.Fatalf("first Finalize: %v", err)
	}
	if _, err := r.Finalize(1); err == nil {
		t.Fatal("expected error on second Finalize")
	}
	if err := r.Add(2, []byte("BBBB")); err == nil {
		t.Fatal("expected error on Add after Finalize")
	}
}

// pending returns a copy of the drained output so tests can inspect progress
// without finalizing. It is only used by tests.
func (r *Reassembly) pending() []byte {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]byte, len(r.final))
	copy(out, r.final)
	return out
}

// TestReassemblyInvalidSeq verifies that non-positive sequence numbers are
// rejected.
func TestReassemblyInvalidSeq(t *testing.T) {
	r := NewReassembly()
	for _, seq := range []int{0, -1, -5} {
		if err := r.Add(seq, []byte("X")); err == nil {
			t.Errorf("Add(%d): expected error, got nil", seq)
		}
	}
}
