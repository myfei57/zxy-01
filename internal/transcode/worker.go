package transcode

import (
	"fmt"

	"edge-transcode/internal/segment"
)

// SegmentCount is how many output segments one source is split into.
const SegmentCount = 5

// Worker splits a source into numbered output segments and writes them to
// the segment store. A write failure is returned so the job can fail
// instead of reporting success over missing output.
type Worker struct {
	store *segment.Store
}

// NewWorker creates a transcode worker.
func NewWorker(store *segment.Store) *Worker {
	return &Worker{store: store}
}

// Transcode splits source bytes into SegmentCount output segments and
// returns their ids in sequence order.
func (w *Worker) Transcode(streamID string, generation int64, source []byte) ([]string, error) {
	ids := make([]string, 0, SegmentCount)
	size := len(source) / SegmentCount
	if size < 1 {
		size = 1
	}
	for seq := 1; seq <= SegmentCount; seq++ {
		start := (seq - 1) * size
		end := start + size
		if seq == SegmentCount || end > len(source) {
			end = len(source)
		}
		var data []byte
		if start < len(source) {
			data = source[start:end]
		}
		id := fmt.Sprintf("%s-seg-%02d", streamID, seq)
		// A storage failure here must surface as a job error: if the
		// bytes never landed, the segment id would be silently listed
		// as produced and the gap only surfaces at manifest time.
		if _, err := w.store.Put(id, seq, generation, data); err != nil {
			return nil, fmt.Errorf("store segment %s: %w", id, err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}
