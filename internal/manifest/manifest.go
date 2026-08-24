// Package manifest builds and publishes playback playlists from the
// contiguous segment sequence of a stream.
package manifest

// Manifest is one immutable playlist revision.
type Manifest struct {
	StreamID   string
	Revision   int64
	Generation int64
	SegmentIDs []string
	Published  bool
}

// New creates an unpublished manifest revision.
func New(streamID string, revision, generation int64, segmentIDs []string) *Manifest {
	return &Manifest{
		StreamID:   streamID,
		Revision:   revision,
		Generation: generation,
		SegmentIDs: segmentIDs,
	}
}
