package manifest

import (
	"strings"
	"testing"

	"edge-transcode/internal/segment"
)

// seedStore fills a segment store with one-byte segments for the given
// sequence numbers, all in one generation.
func seedStore(t *testing.T, dir string, generation int64, seqs ...int) *segment.Store {
	t.Helper()
	st := segment.NewStore(dir)
	for _, seq := range seqs {
		id := idForSeq(seq)
		if _, err := st.Put(id, seq, generation, []byte{byte(seq)}); err != nil {
			t.Fatalf("put seq %d: %v", seq, err)
		}
	}
	return st
}

func idForSeq(seq int) string {
	ids := []string{"", "seg-01", "seg-02", "seg-03", "seg-04", "seg-05"}
	if seq >= 1 && seq < len(ids) {
		return ids[seq]
	}
	return "seg-other"
}

// TestBuildContiguous confirms a complete 1..max sequence builds a manifest
// referencing every segment id, in order.
func TestBuildContiguous(t *testing.T) {
	st := seedStore(t, t.TempDir(), 1, 1, 2, 3, 4, 5)
	b := NewBuilder(st)

	ids, err := b.Build("stream", 1, 5)
	if err != nil {
		t.Fatalf("expected build to succeed, got %v", err)
	}
	want := []string{"seg-01", "seg-02", "seg-03", "seg-04", "seg-05"}
	if len(ids) != len(want) {
		t.Fatalf("got %v, want %v", ids, want)
	}
	for i, id := range ids {
		if id != want[i] {
			t.Fatalf("seq %d: got %s want %s", i+1, id, want[i])
		}
	}
}

// TestBuildGapStopsPublication is the regression for the reported defect.
// With sequence 3 missing, Build must surface an error instead of producing
// a manifest that references a gap. Before the fix the OrderedSegments error
// was discarded and the manifest advanced over the hole, surfacing as 404 /
// "segment not found" at playback.
func TestBuildGapStopsPublication(t *testing.T) {
	// segments 1,2,4,5 present — seq 3 is the gap.
	st := seedStore(t, t.TempDir(), 1, 1, 2, 4, 5)
	b := NewBuilder(st)

	ids, err := b.Build("stream", 1, 5)
	if err == nil {
		t.Fatalf("expected error on missing seq 3, got nil with ids=%v", ids)
	}
	if len(ids) != 0 {
		t.Fatalf("expected no ids on gap, got %v", ids)
	}
	if !strings.Contains(err.Error(), "missing segment sequence 3") {
		t.Fatalf("expected gap error to mention seq 3, got %v", err)
	}
}

// TestBuildGenerationMismatchStopsPublication confirms a generation mismatch
// (stale segments from a prior generation) is treated as an inconsistency
// and blocks publication — the manifest must not mix generations.
func TestBuildGenerationMismatchStopsPublication(t *testing.T) {
	st := seedStore(t, t.TempDir(), 1, 1, 2, 3, 4, 5)
	b := NewBuilder(st)

	// ask for generation 2 — none of the segments belong to it.
	ids, err := b.Build("stream", 2, 5)
	if err == nil {
		t.Fatalf("expected error on generation mismatch, got nil with ids=%v", ids)
	}
	if len(ids) != 0 {
		t.Fatalf("expected no ids on generation mismatch, got %v", ids)
	}
}
