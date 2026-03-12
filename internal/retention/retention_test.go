package retention

import (
	"testing"
	"time"
)

func TestParseSearchKeys(t *testing.T) {
	result := []any{int64(3), "doc:1", "doc:2", "doc:3"}

	keys := parseSearchKeys(result)
	if len(keys) != 3 {
		t.Fatalf("expected 3 keys, got %d", len(keys))
	}
	if keys[0] != "doc:1" {
		t.Errorf("keys[0] = %q, want %q", keys[0], "doc:1")
	}
}

func TestParseSearchKeysEmpty(t *testing.T) {
	result := []any{int64(0)}
	keys := parseSearchKeys(result)
	if len(keys) != 0 {
		t.Fatalf("expected 0 keys, got %d", len(keys))
	}
}

func TestParseSearchKeysNil(t *testing.T) {
	keys := parseSearchKeys(nil)
	if keys != nil {
		t.Fatalf("expected nil, got %v", keys)
	}
}

func TestNewCleaner(t *testing.T) {
	c := New(nil, "test-index", 168*time.Hour, time.Hour)
	if c.index != "test-index" {
		t.Errorf("index = %q, want %q", c.index, "test-index")
	}
	if c.stream != "test-index" {
		t.Errorf("stream = %q, want %q (should default to index)", c.stream, "test-index")
	}
	if c.ttl != 168*time.Hour {
		t.Errorf("ttl = %v, want %v", c.ttl, 168*time.Hour)
	}
	if c.interval != time.Hour {
		t.Errorf("interval = %v, want %v", c.interval, time.Hour)
	}
}

func TestParseSearchEntries(t *testing.T) {
	// FT.SEARCH with RETURN 1 timestamp response:
	// [2, "key1", ["timestamp", "2026-03-11T00:30:06Z"], "key2", ["timestamp", "2026-03-12T10:00:00Z"]]
	result := []any{
		int64(2),
		"key1", []any{"timestamp", "2026-03-11T00:30:06Z"},
		"key2", []any{"timestamp", "2026-03-12T10:00:00Z"},
	}

	entries := parseSearchEntries(result)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].key != "key1" {
		t.Errorf("entries[0].key = %q, want %q", entries[0].key, "key1")
	}
	if entries[0].timestamp != "2026-03-11T00:30:06Z" {
		t.Errorf("entries[0].timestamp = %q, want %q", entries[0].timestamp, "2026-03-11T00:30:06Z")
	}
	if entries[1].key != "key2" {
		t.Errorf("entries[1].key = %q, want %q", entries[1].key, "key2")
	}
}

func TestParseSearchEntriesEmpty(t *testing.T) {
	result := []any{int64(0)}
	entries := parseSearchEntries(result)
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(entries))
	}
}

func TestParseSearchEntriesNil(t *testing.T) {
	entries := parseSearchEntries(nil)
	if entries != nil {
		t.Fatalf("expected nil, got %v", entries)
	}
}

func TestStreamMinIDFormat(t *testing.T) {
	// Verify that UnixMilli produces a valid millisecond timestamp for stream IDs
	cutoff := time.Date(2026, 3, 11, 0, 0, 0, 0, time.UTC)
	minID := cutoff.UnixMilli()

	// Should be positive and in milliseconds (13 digits for 2026)
	if minID <= 0 {
		t.Errorf("expected positive millisecond timestamp, got %d", minID)
	}
	// Round-trip: convert back and verify
	roundTrip := time.UnixMilli(minID).UTC()
	if !roundTrip.Equal(cutoff) {
		t.Errorf("round-trip mismatch: got %v, want %v", roundTrip, cutoff)
	}
}
