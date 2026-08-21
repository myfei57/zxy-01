package console

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"edge-transcode/internal/ingest"
	"edge-transcode/internal/playback"
	"edge-transcode/internal/publish"
	"edge-transcode/internal/segment"
	"edge-transcode/internal/store"
	"edge-transcode/internal/transcode"
)

// API is the HTTP surface of the service.
type API struct {
	state        *store.State
	uploads      *ingest.Handler
	orchestrator *transcode.Orchestrator
	origin       *playback.Origin
	cache        *playback.Cache
	cdn          *publish.CDN
	segmentStore *segment.Store
}

// NewAPI wires the console API.
func NewAPI(
	state *store.State,
	uploads *ingest.Handler,
	orchestrator *transcode.Orchestrator,
	origin *playback.Origin,
	cache *playback.Cache,
	cdn *publish.CDN,
	segmentStore *segment.Store,
) *API {
	return &API{
		state:        state,
		uploads:      uploads,
		orchestrator: orchestrator,
		origin:       origin,
		cache:        cache,
		cdn:          cdn,
		segmentStore: segmentStore,
	}
}

// Router builds the full HTTP route tree.
func (a *API) Router() http.Handler {
	r := chi.NewRouter()
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		write(w, http.StatusOK, "ok")
	})
	r.Mount("/", a.uploads.Routes())
	r.Get("/api/streams", a.listStreams)
	r.Get("/api/jobs", a.listJobs)
	r.Get("/api/quality", a.quality)
	r.Get("/api/nodes", a.nodes)
	r.Post("/api/streams/{id}/process", a.process)
	r.Get("/api/play/{streamID}/{seq}", a.play)
	r.Get("/", a.index)
	r.Get("/console/tasks", a.tasksPage)
	r.Get("/console/streams", a.streamsPage)
	r.Get("/console/nodes", a.nodesPage)
	r.Get("/console/quality", a.qualityPage)
	return r
}

func (a *API) listStreams(w http.ResponseWriter, _ *http.Request) {
	type streamView struct {
		ID       string  `json:"id"`
		Title    string  `json:"title"`
		Owner    string  `json:"owner"`
		State    string  `json:"state"`
		Segments int     `json:"segments"`
		Revision int64   `json:"revision"`
		Success  float64 `json:"success_rate"`
		Average  float64 `json:"average_score"`
		Cursor   int     `json:"cursor"`
	}
	samples := a.state.Samples()
	out := make([]streamView, 0)
	for _, stream := range a.state.ListStreams() {
		revision := int64(0)
		if manifest, ok := a.state.Manifest(stream.ID); ok {
			revision = manifest.Revision
		}
		segments := 0
		for _, segment := range a.state.ListSegments() {
			if segment.StreamID == stream.ID {
				segments++
			}
		}
		out = append(out, streamView{
			ID:       stream.ID,
			Title:    stream.Title,
			Owner:    stream.Owner,
			State:    stream.State,
			Segments: segments,
			Revision: revision,
			Success:  SuccessRate(samples, stream.ID),
			Average:  AverageScore(samples, stream.ID),
			Cursor:   a.state.Cursor(stream.ID),
		})
	}
	writeJSON(w, out)
}

func (a *API) listJobs(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, a.state.ListJobs())
}

func (a *API) quality(w http.ResponseWriter, _ *http.Request) {
	samples := a.state.Samples()
	writeJSON(w, map[string]any{
		"pass_count": PassCount(samples, ""),
		"samples":    RecentSamples(samples, 200),
	})
}

func (a *API) nodes(w http.ResponseWriter, _ *http.Request) {
	type nodeView struct {
		Name       string `json:"name"`
		Region     string `json:"region"`
		Segments   int    `json:"segments"`
		CacheHits  int    `json:"cache_entries"`
		CDNObjects int    `json:"cdn_objects"`
	}
	segments := len(a.state.ListSegments())
	writeJSON(w, []nodeView{
		{Name: "edge-sh-01", Region: "east", Segments: segments, CacheHits: a.cache.Count(), CDNObjects: a.cdn.Count()},
		{Name: "edge-gz-01", Region: "south", Segments: segments, CacheHits: a.cache.Count(), CDNObjects: a.cdn.Count()},
		{Name: "edge-bj-01", Region: "north", Segments: segments, CacheHits: a.cache.Count(), CDNObjects: a.cdn.Count()},
	})
}

func (a *API) process(w http.ResponseWriter, r *http.Request) {
	streamID := chi.URLParam(r, "id")
	if err := a.orchestrator.ProcessStream(streamID); err != nil {
		_ = a.state.Save()
		writeJSONStatus(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}
	_ = a.state.Save()
	writeJSON(w, map[string]string{"stream_id": streamID, "state": "live"})
}

func (a *API) play(w http.ResponseWriter, r *http.Request) {
	streamID := chi.URLParam(r, "streamID")
	seq := chi.URLParam(r, "seq")
	key := fmt.Sprintf("%s-seg-%s", streamID, seq)
	data, status, err := a.origin.Serve(key)
	if err != nil {
		writeJSONStatus(w, status, map[string]string{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "video/mp2t")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func write(w http.ResponseWriter, status int, body string) {
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}

func writeJSON(w http.ResponseWriter, value any) {
	writeJSONStatus(w, http.StatusOK, value)
}

func writeJSONStatus(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
