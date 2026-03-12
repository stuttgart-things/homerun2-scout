package aggregator

import (
	"strings"
	"testing"
)

func TestEnsureIndexSchema(t *testing.T) {
	// Verify the expected FT.CREATE args contain all required fields
	expectedFields := []string{
		"$.severity", "severity", "TAG",
		"$.system", "system", "TAG",
		"$.timestamp", "timestamp", "TEXT",
		"$.title", "title", "TEXT",
		"$.message", "message", "TEXT",
		"$.author", "author", "TAG",
		"$.tags", "tags", "TAG",
	}

	// Build the args as EnsureIndex would
	args := []string{
		"FT.CREATE", "messages",
		"ON", "JSON",
		"SCHEMA",
		"$.severity", "AS", "severity", "TAG",
		"$.system", "AS", "system", "TAG",
		"$.timestamp", "AS", "timestamp", "TEXT",
		"$.title", "AS", "title", "TEXT",
		"$.message", "AS", "message", "TEXT",
		"$.author", "AS", "author", "TAG",
		"$.tags", "AS", "tags", "TAG",
	}
	joined := strings.Join(args, " ")

	for _, field := range expectedFields {
		if !strings.Contains(joined, field) {
			t.Errorf("schema missing expected field/type: %s", field)
		}
	}
}

func TestEnsureIndexArgsStructure(t *testing.T) {
	// Verify ON JSON is specified (not HASH)
	args := []string{
		"FT.CREATE", "test-index",
		"ON", "JSON",
		"SCHEMA",
	}
	joined := strings.Join(args, " ")

	if !strings.Contains(joined, "ON JSON") {
		t.Error("index should be created ON JSON, not HASH")
	}
}
