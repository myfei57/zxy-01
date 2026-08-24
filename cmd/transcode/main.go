// Command transcode runs the EdgeTranscode service.
package main

import (
	"flag"
	"log"
	"net/http"
	"path/filepath"

	"edge-transcode/internal/console"
	"edge-transcode/internal/ingest"
	"edge-transcode/internal/manifest"
	"edge-transcode/internal/playback"
	"edge-transcode/internal/publish"
	"edge-transcode/internal/qc"
	"edge-transcode/internal/segment"
	"edge-transcode/internal/store"
	"edge-transcode/internal/transcode"
)

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	dataDir := flag.String("data", "data", "data directory")
	flag.Parse()

	state := store.NewState(*dataDir)
	if err := state.Load(); err != nil {
		log.Fatalf("load state: %v", err)
	}
	segmentStore := segment.NewStore(filepath.Join(*dataDir, "segments"))
	checkpoint := store.NewCheckpoint(*dataDir)
	threshold := qc.NewThreshold(45, 65)
	checker := qc.NewChecker(segmentStore, threshold)
	worker := transcode.NewWorker(segmentStore)
	builder := manifest.NewBuilder(segmentStore)
	publisher := manifest.NewPublisher(state, checkpoint)
	cdn := publish.NewCDN()
	cursor := publish.NewCursor()
	batch := publish.NewBatch(cdn, cursor, segmentStore)
	orchestrator := transcode.NewOrchestrator(segmentStore, state, worker, checker, builder, publisher, batch)

	manager := ingest.NewManager()
	uploadService := ingest.NewService(manager, segmentStore)
	uploadHandler := ingest.NewHandler(uploadService, manager, state)

	cache := playback.NewCache()
	origin := playback.NewOrigin(cdn, cache)
	api := console.NewAPI(state, uploadHandler, orchestrator, origin, cache, cdn, segmentStore)

	log.Printf("EdgeTranscode listening on %s (data: %s)", *addr, *dataDir)
	if err := http.ListenAndServe(*addr, api.Router()); err != nil {
		log.Fatal(err)
	}
}
