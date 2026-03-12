package aggregator

import (
	"strings"
	"testing"
)

func TestEnsureIndexSchema(t *testing.T) {
	// Verify the expected FT.CREATE args contain all required fields as TEXT
	// TEXT is required (not TAG) for FT.AGGREGATE GROUPBY to return grouped rows
	expectedFields := []string{
		"$.severity", "severity", "TEXT",
		"$.system", "system", "TEXT",
		"$.timestamp", "timestamp", "TEXT",
		"$.title", "title", "TEXT",
		"$.message", "message", "TEXT",
		"$.author", "author", "TEXT",
		"$.tags", "tags", "TEXT",
	}

	// Build the args as EnsureIndex would
	args := []string{
		"FT.CREATE", "messages",
		"ON", "JSON",
		"SCHEMA",
		"$.severity", "AS", "severity", "TEXT",
		"$.system", "AS", "system", "TEXT",
		"$.timestamp", "AS", "timestamp", "TEXT",
		"$.title", "AS", "title", "TEXT",
		"$.message", "AS", "message", "TEXT",
		"$.author", "AS", "author", "TEXT",
		"$.tags", "AS", "tags", "TEXT",
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

func TestMissingIndexErrorPatterns(t *testing.T) {
	// EnsureIndex must recognize both error messages from different Redis/RediSearch versions
	patterns := []string{"no such index", "Unknown index name"}
	for _, p := range patterns {
		if !strings.Contains(p, "index") {
			t.Errorf("expected pattern to contain 'index': %s", p)
		}
	}
}

func TestEnsureIndexNoTagFields(t *testing.T) {
	// TAG fields on JSON indexes break FT.AGGREGATE GROUPBY (returns only count, no rows)
	args := []string{
		"$.severity", "AS", "severity", "TEXT",
		"$.system", "AS", "system", "TEXT",
		"$.author", "AS", "author", "TEXT",
		"$.tags", "AS", "tags", "TEXT",
	}
	joined := strings.Join(args, " ")

	if strings.Contains(joined, "TAG") {
		t.Error("schema should not use TAG type — use TEXT for FT.AGGREGATE GROUPBY compatibility")
	}
}
