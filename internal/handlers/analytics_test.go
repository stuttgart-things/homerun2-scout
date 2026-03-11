package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stuttgart-things/homerun2-scout/internal/models"
)

type mockProvider struct{}

func (m *mockProvider) Summary() *models.Summary {
	return &models.Summary{
		TotalMessages:  42,
		SeverityCounts: map[string]int64{"info": 30, "error": 12},
		TimeWindow:     "60s",
		LastUpdated:    time.Now(),
	}
}

func (m *mockProvider) Systems() *models.SystemStats {
	return &models.SystemStats{
		Systems: []models.SystemCount{
			{System: "api-gateway", Count: 20},
			{System: "worker", Count: 22},
		},
		Total:       2,
		LastUpdated: time.Now(),
	}
}

func (m *mockProvider) Alerts() *models.AlertStats {
	return &models.AlertStats{
		TotalAlerts:    12,
		SeverityCounts: map[string]int64{"error": 10, "critical": 2},
		TopSystems: []models.SystemCount{
			{System: "api-gateway", Count: 8},
		},
		LastUpdated: time.Now(),
	}
}

func TestSummaryHandler(t *testing.T) {
	handler := NewSummaryHandler(&mockProvider{})
	req := httptest.NewRequest(http.MethodGet, "/analytics/summary", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	var resp models.Summary
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp.TotalMessages != 42 {
		t.Errorf("TotalMessages = %d, want %d", resp.TotalMessages, 42)
	}
}

func TestSystemsHandler(t *testing.T) {
	handler := NewSystemsHandler(&mockProvider{})
	req := httptest.NewRequest(http.MethodGet, "/analytics/systems", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	var resp models.SystemStats
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp.Total != 2 {
		t.Errorf("Total = %d, want %d", resp.Total, 2)
	}
}

func TestAlertsHandler(t *testing.T) {
	handler := NewAlertsHandler(&mockProvider{})
	req := httptest.NewRequest(http.MethodGet, "/analytics/alerts", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	var resp models.AlertStats
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp.TotalAlerts != 12 {
		t.Errorf("TotalAlerts = %d, want %d", resp.TotalAlerts, 12)
	}
}

func TestSummaryHandler_MethodNotAllowed(t *testing.T) {
	handler := NewSummaryHandler(&mockProvider{})
	req := httptest.NewRequest(http.MethodPost, "/analytics/summary", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusMethodNotAllowed)
	}
}
