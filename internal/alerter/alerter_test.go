package alerter

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stuttgart-things/homerun2-scout/internal/models"
)

func TestCheck_ErrorThresholdExceeded(t *testing.T) {
	var received atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received.Add(1)
		var req PitchRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode error: %v", err)
		}
		if req.Severity != "WARNING" {
			t.Errorf("severity = %q, want %q", req.Severity, "WARNING")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	a := New(server.URL, "test-token", ThresholdConfig{
		ErrorThreshold: 10,
		Cooldown:       time.Second,
	})

	summary := &models.Summary{
		SeverityCounts: map[string]int64{"error": 15},
	}
	alerts := &models.AlertStats{}

	a.Check(context.Background(), summary, alerts)

	if received.Load() != 1 {
		t.Errorf("expected 1 alert sent, got %d", received.Load())
	}
}

func TestCheck_BelowThreshold(t *testing.T) {
	var received atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	a := New(server.URL, "", ThresholdConfig{
		ErrorThreshold:    100,
		CriticalThreshold: 10,
		Cooldown:          time.Second,
	})

	summary := &models.Summary{
		SeverityCounts: map[string]int64{"error": 5, "critical": 1},
	}
	alerts := &models.AlertStats{TotalAlerts: 6}

	a.Check(context.Background(), summary, alerts)

	if received.Load() != 0 {
		t.Errorf("expected 0 alerts sent, got %d", received.Load())
	}
}

func TestCheck_CooldownSuppression(t *testing.T) {
	var received atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	a := New(server.URL, "", ThresholdConfig{
		ErrorThreshold: 5,
		Cooldown:       time.Hour, // Long cooldown
	})

	summary := &models.Summary{
		SeverityCounts: map[string]int64{"error": 10},
	}
	alerts := &models.AlertStats{}

	// First call sends
	a.Check(context.Background(), summary, alerts)
	// Second call suppressed by cooldown
	a.Check(context.Background(), summary, alerts)

	if received.Load() != 1 {
		t.Errorf("expected 1 alert (2nd suppressed by cooldown), got %d", received.Load())
	}
}

// The aggregator groups by the severity as pitched, so ERROR and error are
// separate counts; both must add up to the threshold. On homerun2-test1 scout
// counted ERROR 1 and error 1 side by side.
func TestCheck_SeverityInAnyCase(t *testing.T) {
	var titles []string
	var mu sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req PitchRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode error: %v", err)
		}
		mu.Lock()
		titles = append(titles, req.Title)
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	a := New(server.URL, "", ThresholdConfig{ErrorThreshold: 10, CriticalThreshold: 3, Cooldown: time.Second})
	summary := &models.Summary{
		SeverityCounts: map[string]int64{"ERROR": 7, "error": 2, "Error": 1, "CRITICAL": 3, "INFO": 500},
	}
	a.Check(context.Background(), summary, &models.AlertStats{})

	want := []string{"Scout: Error threshold exceeded", "Scout: Critical threshold exceeded"}
	if len(titles) != len(want) || titles[0] != want[0] || titles[1] != want[1] {
		t.Errorf("alerts = %q, want %q", titles, want)
	}
}

func TestSeverityCount(t *testing.T) {
	counts := map[string]int64{"ERROR": 4, "error": 1, "critical": 2}
	cases := []struct {
		severity string
		want     int64
		found    bool
	}{
		{"error", 5, true},
		{"ERROR", 5, true},
		{"critical", 2, true},
		{"warning", 0, false},
	}
	for _, tc := range cases {
		if got, found := severityCount(counts, tc.severity); got != tc.want || found != tc.found {
			t.Errorf("severityCount(%q) = %d, %v; want %d, %v", tc.severity, got, found, tc.want, tc.found)
		}
	}
}

func TestCheck_NoPitcherURL(t *testing.T) {
	a := New("", "", ThresholdConfig{ErrorThreshold: 1})
	summary := &models.Summary{
		SeverityCounts: map[string]int64{"error": 100},
	}
	// Should not panic
	a.Check(context.Background(), summary, &models.AlertStats{})
}
