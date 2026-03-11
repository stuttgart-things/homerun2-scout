package retention

import (
	"testing"
	"time"
)

func TestParseSearchKeys(t *testing.T) {
	// FT.SEARCH NOCONTENT response: [3, "key1", "key2", "key3"]
	result := []interface{}{int64(3), "doc:1", "doc:2", "doc:3"}

	keys := parseSearchKeys(result)
	if len(keys) != 3 {
		t.Fatalf("expected 3 keys, got %d", len(keys))
	}
	if keys[0] != "doc:1" {
		t.Errorf("keys[0] = %q, want %q", keys[0], "doc:1")
	}
}

func TestParseSearchKeysEmpty(t *testing.T) {
	result := []interface{}{int64(0)}
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
	if c.ttl != 168*time.Hour {
		t.Errorf("ttl = %v, want %v", c.ttl, 168*time.Hour)
	}
	if c.interval != time.Hour {
		t.Errorf("interval = %v, want %v", c.interval, time.Hour)
	}
}
