package verifycase

import (
	"testing"

	"edge-transcode/internal/manifest"
	"edge-transcode/internal/segment"
)

// TestManifestRejectsMissingSegmentSequence verifies a manifest never
// advances over a gap in the segment sequence.
func TestManifestRejectsMissingSegmentSequence(t *testing.T) {
	dir := t.TempDir()
	st := segment.NewStore(dir)
	if _, err := st.Put("s1", 1, 7, []byte("one")); err != nil {
		t.Fatal(err)
	}
	if _, err := st.Put("s2", 2, 7, []byte("two")); err != nil {
		t.Fatal(err)
	}
	builder := manifest.NewBuilder(st)
	if _, err := builder.Build("stream", 7, 3); err == nil {
		t.Fatal("manifest built over a missing sequence; expected error")
	}
}
