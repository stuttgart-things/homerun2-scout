package aggregator

import (
	"testing"
	"time"
)

func TestParseAggregateResult(t *testing.T) {
	// Simulate FT.AGGREGATE response: [2, ["severity", "info", "count", "10"], ["severity", "error", "count", "3"]]
	result := []interface{}{
		int64(2),
		[]interface{}{"severity", "info", "count", "10"},
		[]interface{}{"severity", "error", "count", "3"},
	}

	rows := parseAggregateResult(result)
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}

	if rows[0]["severity"] != "info" {
		t.Errorf("rows[0][severity] = %q, want %q", rows[0]["severity"], "info")
	}
	if rows[0]["count"] != "10" {
		t.Errorf("rows[0][count] = %q, want %q", rows[0]["count"], "10")
	}
	if rows[1]["severity"] != "error" {
		t.Errorf("rows[1][severity] = %q, want %q", rows[1]["severity"], "error")
	}
	if rows[1]["count"] != "3" {
		t.Errorf("rows[1][count] = %q, want %q", rows[1]["count"], "3")
	}
}

func TestParseAggregateResultEmpty(t *testing.T) {
	result := []interface{}{int64(0)}
	rows := parseAggregateResult(result)
	if len(rows) != 0 {
		t.Fatalf("expected 0 rows, got %d", len(rows))
	}
}

func TestParseAggregateResultNil(t *testing.T) {
	rows := parseAggregateResult(nil)
	if rows != nil {
		t.Fatalf("expected nil, got %v", rows)
	}
}

func TestNewAggregator(t *testing.T) {
	agg := New(nil, "test-index", 30*time.Second)
	if agg.index != "test-index" {
		t.Errorf("index = %q, want %q", agg.index, "test-index")
	}
	if agg.interval != 30*time.Second {
		t.Errorf("interval = %v, want %v", agg.interval, 30*time.Second)
	}
}

func TestSummaryDefault(t *testing.T) {
	agg := New(nil, "test", time.Minute)
	s := agg.Summary()
	if s == nil {
		t.Fatal("Summary() returned nil")
	}
	if s.SeverityCounts == nil {
		t.Error("SeverityCounts should not be nil")
	}
}

func TestSystemsDefault(t *testing.T) {
	agg := New(nil, "test", time.Minute)
	s := agg.Systems()
	if s == nil {
		t.Fatal("Systems() returned nil")
	}
	if s.Systems == nil {
		t.Error("Systems should not be nil")
	}
}

func TestAlertsDefault(t *testing.T) {
	agg := New(nil, "test", time.Minute)
	a := agg.Alerts()
	if a == nil {
		t.Fatal("Alerts() returned nil")
	}
	if a.SeverityCounts == nil {
		t.Error("SeverityCounts should not be nil")
	}
}
