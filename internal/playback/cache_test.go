package playback

import (
	"bytes"
	"errors"
	"testing"
)

// TestWriteRefusesNonSuccess verifies that a failed origin fetch (5xx) can
// never poison the cache: non-success statuses are refused, the entry is
// not written, and any existing good entry is left untouched.
func TestWriteRefusesNonSuccess(t *testing.T) {
	cases := []int{100, 199, 300, 301, 404, 500, 502, 503, 504}
	for _, status := range cases {
		c := NewCache()
		err := c.Write("seg-1", status, []byte("error body"))
		if !errors.Is(err, ErrNonSuccess) {
			t.Errorf("status %d: expected ErrNonSuccess, got %v", status, err)
		}
		if _, ok := c.Lookup("seg-1"); ok {
			t.Errorf("status %d: error response must not be cached", status)
		}
		if c.Count() != 0 {
			t.Errorf("status %d: cache must remain empty, count=%d", status, c.Count())
		}
	}
}

// TestWriteAcceptsSuccess verifies that only 2xx responses are stored.
func TestWriteAcceptsSuccess(t *testing.T) {
	for _, status := range []int{200, 201, 204, 299} {
		c := NewCache()
		if err := c.Write("seg-1", status, []byte("good")); err != nil {
			t.Fatalf("status %d: unexpected error: %v", status, err)
		}
		entry, ok := c.Lookup("seg-1")
		if !ok {
			t.Fatalf("status %d: expected entry to be cached", status)
		}
		if entry.Status != status {
			t.Errorf("status %d: cached status %d", status, entry.Status)
		}
		if !bytes.Equal(entry.Data, []byte("good")) {
			t.Errorf("status %d: cached data mismatch", status)
		}
	}
}

// TestWriteDoesNotClobberGoodEntryOnFailure verifies that a subsequent error
// response cannot overwrite a previously cached good entry, so a transient
// origin failure cannot stick as stale bad content that survives refreshes.
func TestWriteDoesNotClobberGoodEntryOnFailure(t *testing.T) {
	c := NewCache()
	if err := c.Write("seg-1", 200, []byte("good")); err != nil {
		t.Fatalf("seed write failed: %v", err)
	}
	if err := c.Write("seg-1", 503, []byte("bad")); !errors.Is(err, ErrNonSuccess) {
		t.Fatalf("expected ErrNonSuccess on 503, got %v", err)
	}
	entry, ok := c.Lookup("seg-1")
	if !ok {
		t.Fatal("good entry must survive a failed write")
	}
	if !bytes.Equal(entry.Data, []byte("good")) {
		t.Fatalf("good entry was clobbered: %q", entry.Data)
	}
	if entry.Status != 200 {
		t.Fatalf("good entry status changed: %d", entry.Status)
	}
}
