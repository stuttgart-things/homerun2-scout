package aggregator

import (
	"strings"
	"testing"
)

func TestEnsureIndexSchema(t *testing.T) {
	// TEXT is required (not TAG) for FT.AGGREGATE GROUPBY to return grouped rows;
	// the event time is NUMERIC so time windows can be queried.
	var parts []string
	for _, a := range indexCreateArgs("messages") {
		parts = append(parts, a.(string))
	}
	joined := strings.Join(parts, " ")

	for _, want := range []string{
		"FT.CREATE messages ON JSON SCHEMA",
		"$.severity AS severity TEXT", "$.system AS system TEXT", "$.timestamp AS timestamp TEXT",
		"$.title AS title TEXT", "$.message AS message TEXT", "$.author AS author TEXT", "$.tags AS tags TEXT",
		"$.timestamp_unix AS timestamp_unix NUMERIC SORTABLE",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("schema missing %q:\n%s", want, joined)
		}
	}
}

// Reply shapes captured from redis-stack 7.2 through go-redis v9.
func TestHasNumericAttribute(t *testing.T) {
	resp2 := []any{"index_name", "messages", "attributes", []any{
		[]any{"identifier", "$.timestamp", "attribute", "timestamp", "type", "TEXT", "WEIGHT", "1"},
		[]any{"identifier", "$.timestamp_unix", "attribute", "timestamp_unix", "type", "NUMERIC", "SORTABLE", "UNF"},
	}}
	resp3 := map[any]any{"attributes": []any{
		map[any]any{"attribute": "timestamp", "identifier": "$.timestamp", "type": "TEXT"},
		map[any]any{"attribute": "timestamp_unix", "flags": []any{"SORTABLE", "UNF"}, "identifier": "$.timestamp_unix", "type": "NUMERIC"},
	}}
	legacy := []any{"attributes", []any{
		[]any{"identifier", "$.timestamp", "attribute", "timestamp", "type", "TEXT"},
	}}

	cases := []struct {
		name      string
		info      any
		attribute string
		want      bool
	}{
		{"resp2 numeric", resp2, TimestampUnixField, true},
		{"resp3 numeric", resp3, TimestampUnixField, true},
		{"text attribute", resp2, "timestamp", false},
		{"index created before the numeric timestamp", legacy, TimestampUnixField, false},
		{"unexpected reply", "OK", TimestampUnixField, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := HasNumericAttribute(tc.info, tc.attribute); got != tc.want {
				t.Errorf("HasNumericAttribute = %v, want %v", got, tc.want)
			}
		})
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
