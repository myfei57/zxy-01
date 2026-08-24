package ingest

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"

	"edge-transcode/internal/store"
)

// Handler exposes the upload API over HTTP.
type Handler struct {
	service *Service
	manager *Manager
	state   *store.State
}

// NewHandler creates the upload HTTP handler.
func NewHandler(service *Service, manager *Manager, state *store.State) *Handler {
	return &Handler{service: service, manager: manager, state: state}
}

// Routes mounts the upload endpoints on the router.
func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/uploads", h.createSession)
	mux.HandleFunc("POST /api/uploads/{id}/chunks", h.receiveChunk)
	mux.HandleFunc("POST /api/uploads/{id}/complete", h.completeUpload)
	mux.HandleFunc("DELETE /api/uploads/{id}", h.abortUpload)
	return mux
}

type createRequest struct {
	Title      string `json:"title"`
	Owner      string `json:"owner"`
	ChunkCount int    `json:"chunk_count"`
}

func (h *Handler) createSession(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if req.Title == "" || req.Owner == "" || req.ChunkCount < 1 {
		writeError(w, http.StatusBadRequest, errors.New("title, owner and chunk_count are required"))
		return
	}
	streamID := uuid.NewString()
	sessionID := uuid.NewString()
	stream := &store.Stream{
		ID:         streamID,
		Title:      req.Title,
		Owner:      req.Owner,
		State:      store.StreamDraft,
		Generation: 1,
		CreatedAt:  nowUTC(),
	}
	h.state.CreateStream(stream)
	h.manager.Create(sessionID, streamID, req.ChunkCount)
	writeJSON(w, http.StatusCreated, map[string]any{
		"stream_id":   streamID,
		"session_id":  sessionID,
		"chunk_count": req.ChunkCount,
	})
}

func (h *Handler) receiveChunk(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")
	seq, err := strconv.Atoi(r.URL.Query().Get("seq"))
	if err != nil || seq < 1 {
		writeError(w, http.StatusBadRequest, errors.New("seq query parameter must be a positive integer"))
		return
	}
	data, err := io.ReadAll(io.LimitReader(r.Body, 8<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := h.service.Receive(sessionID, seq, data); err != nil {
		writeError(w, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"received": seq})
}

func (h *Handler) completeUpload(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")
	session, ok := h.manager.Get(sessionID)
	if !ok {
		writeError(w, http.StatusNotFound, errors.New("upload session not found"))
		return
	}
	stream, ok := h.state.Stream(session.StreamID)
	if !ok {
		writeError(w, http.StatusNotFound, errors.New("stream not found"))
		return
	}
	segmentID, err := h.service.Complete(sessionID, session.StreamID, stream.Generation)
	if err != nil {
		writeError(w, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"segment_id": segmentID})
}

func (h *Handler) abortUpload(w http.ResponseWriter, r *http.Request) {
	h.service.Abort(r.PathValue("id"))
	writeJSON(w, http.StatusOK, map[string]any{"aborted": r.PathValue("id")})
}

func nowUTC() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}
