package playback

import (
	"fmt"

	"edge-transcode/internal/publish"
)

// Origin fetches segments from the delivery network and populates the edge
// cache with successful responses only.
type Origin struct {
	cdn   *publish.CDN
	cache *Cache
}

// NewOrigin creates an origin backed by the CDN and the edge cache.
func NewOrigin(cdn *publish.CDN, cache *Cache) *Origin {
	return &Origin{cdn: cdn, cache: cache}
}

// Fetch returns the content of a segment, writing it into the cache only
// when the source responds successfully.
func (o *Origin) Fetch(key string) ([]byte, error) {
	data, err := o.cdn.Get(key)
	if err != nil {
		return nil, err
	}
	if err := o.cache.Write(key, 200, data); err != nil {
		return nil, fmt.Errorf("origin fetch %s: %w", key, err)
	}
	return data, nil
}

// Serve handles one playback request: cache first, origin on miss.
func (o *Origin) Serve(key string) ([]byte, int, error) {
	if entry, ok := o.cache.Lookup(key); ok {
		return entry.Data, entry.Status, nil
	}
	data, err := o.Fetch(key)
	if err != nil {
		return nil, 502, err
	}
	return data, 200, nil
}
