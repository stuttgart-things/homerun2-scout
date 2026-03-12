package alerter

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

func TestCheck_NoPitcherURL(t *testing.T) {
	a := New("", "", ThresholdConfig{ErrorThreshold: 1})
	summary := &models.Summary{
		SeverityCounts: map[string]int64{"error": 100},
	}
	// Should not panic
	a.Check(context.Background(), summary, &models.AlertStats{})
}
