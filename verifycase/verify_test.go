package verifycase

import (
	"testing"

	"edge-transcode/internal/playback"
)

// TestOriginErrorNeverCachedAsSuccess verifies error responses never enter
// the edge cache.
func TestOriginErrorNeverCachedAsSuccess(t *testing.T) {
	cache := playback.NewCache()
	if err := cache.Write("key", 500, []byte("err")); err == nil {
		t.Fatal("error response must not be cached")
	}
}
