package transcode

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"edge-transcode/internal/manifest"
	"edge-transcode/internal/publish"
	"edge-transcode/internal/qc"
	"edge-transcode/internal/segment"
	"edge-transcode/internal/store"
)

// Orchestrator runs the end-to-end pipeline for a stream: transcode, QC,
// manifest build, publish. Failed QC retries the transcode up to
// maxAttempts, then fails the job.
type Orchestrator struct {
	store       *segment.Store
	state       *store.State
	worker      *Worker
	checker     *qc.Checker
	builder     *manifest.Builder
	publisher   *manifest.Publisher
	batch       *publish.Batch
	maxAttempts int
}

// NewOrchestrator wires the pipeline together.
func NewOrchestrator(
	store *segment.Store,
	state *store.State,
	worker *Worker,
	checker *qc.Checker,
	builder *manifest.Builder,
	publisher *manifest.Publisher,
	batch *publish.Batch,
) *Orchestrator {
	return &Orchestrator{
		store:       store,
		state:       state,
		worker:      worker,
		checker:     checker,
		builder:     builder,
		publisher:   publisher,
		batch:       batch,
		maxAttempts: 3,
	}
}

// ProcessStream transcodes, quality-gates, builds and publishes a stream.
func (o *Orchestrator) ProcessStream(streamID string) error {
	stream, ok := o.state.Stream(streamID)
	if !ok {
		return fmt.Errorf("unknown stream %s", streamID)
	}
	source, err := o.store.Get(streamID + "-src")
	if err != nil {
		return o.fail(stream, nil, err)
	}
	job := &Job{
		ID:        uuid.NewString(),
		StreamID:  streamID,
		SegmentID: streamID + "-src",
		Profile:   "adaptive",
		State:     StateQueued,
	}
	stream.State = store.StreamStaging
	o.state.UpdateStream(stream)
	o.persist(job)

	attempts := 0
	for {
		attempts++
		job.Attempts = attempts
		if err := job.Next(StateTranscoding); err != nil {
			return o.fail(stream, job, err)
		}
		o.persist(job)
		segmentIDs, err := o.worker.Transcode(streamID, stream.Generation, source)
		if err != nil {
			return o.fail(stream, job, err)
		}
		if err := job.Next(StateQC); err != nil {
			return o.fail(stream, job, err)
		}
		o.persist(job)
		passed, err := o.qcSegments(streamID, stream.Generation, segmentIDs)
		if err != nil {
			return o.fail(stream, job, err)
		}
		if !passed {
			if attempts >= o.maxAttempts {
				return o.fail(stream, job, errors.New("QC failed after maximum attempts"))
			}
			if err := job.Next(StateQueued); err != nil {
				return o.fail(stream, job, err)
			}
			o.persist(job)
			continue
		}
		if err := job.Next(StatePublishing); err != nil {
			return o.fail(stream, job, err)
		}
		o.persist(job)
		ordered, err := o.builder.Build(streamID, stream.Generation, SegmentCount)
		if err != nil {
			return o.fail(stream, job, err)
		}
		revision, err := o.publisher.RecoveryRevision(streamID)
		if err != nil {
			return o.fail(stream, job, err)
		}
		if err := o.publisher.Publish(manifest.New(streamID, revision+1, stream.Generation, ordered)); err != nil {
			return o.fail(stream, job, err)
		}
		if err := o.batch.Push(streamID, ordered); err != nil {
			return o.fail(stream, job, err)
		}
		o.markPublished(streamID, stream.Generation, ordered)
		o.state.SetCursor(streamID, len(ordered))
		if err := job.Next(StateDone); err != nil {
			return o.fail(stream, job, err)
		}
		o.persist(job)
		stream.State = store.StreamLive
		o.state.UpdateStream(stream)
		return nil
	}
}

// qcSegments evaluates every output segment and durably marks the passed
// ones. It returns false when any verdict is failed.
func (o *Orchestrator) qcSegments(streamID string, generation int64, ids []string) (bool, error) {
	o.checker.Reset()
	for _, id := range ids {
		data, err := o.store.Get(id)
		if err != nil {
			return false, err
		}
		score := o.checker.Score(data)
		verdict := o.checker.Evaluate(score)
		o.state.AddSample(store.QualitySample{
			StreamID:  streamID,
			SegmentID: id,
			Score:     score,
			Passed:    verdict.Passed,
		})
		if !verdict.Passed {
			return false, nil
		}
		if err := o.checker.Pass(id, data); err != nil {
			return false, err
		}
		meta, _ := o.store.Meta(id)
		o.state.PutSegment(&store.Segment{
			ID:         id,
			StreamID:   streamID,
			Seq:        meta.Seq,
			Size:       meta.Size,
			State:      store.SegmentQCPassed,
			Generation: generation,
			Digest:     store.Digest(data),
		})
	}
	return true, nil
}

func (o *Orchestrator) markPublished(streamID string, generation int64, ids []string) {
	for _, id := range ids {
		meta, ok := o.store.Meta(id)
		if !ok {
			continue
		}
		o.state.PutSegment(&store.Segment{
			ID:         id,
			StreamID:   streamID,
			Seq:        meta.Seq,
			Size:       meta.Size,
			State:      store.SegmentPublished,
			Generation: generation,
		})
	}
}

func (o *Orchestrator) persist(job *Job) {
	if job.CreatedAt == "" {
		job.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	o.state.PutJob(&store.Job{
		ID:        job.ID,
		StreamID:  job.StreamID,
		SegmentID: job.SegmentID,
		Profile:   job.Profile,
		State:     job.State,
		Attempts:  job.Attempts,
		Error:     job.Error,
		CreatedAt: job.CreatedAt,
	})
}

func (o *Orchestrator) fail(stream *store.Stream, job *Job, err error) error {
	stream.State = store.StreamFailed
	o.state.UpdateStream(stream)
	if job != nil {
		job.Fail(err)
		o.persist(job)
	}
	return err
}
