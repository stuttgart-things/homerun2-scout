package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/stuttgart-things/homerun2-scout/internal/models"
)

// AnalyticsProvider defines the interface for retrieving analytics data.
type AnalyticsProvider interface {
	Summary() *models.Summary
	Systems() *models.SystemStats
	Alerts() *models.AlertStats
}

// NewSummaryHandler returns a handler for GET /analytics/summary.
func NewSummaryHandler(provider AnalyticsProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}
		respondJSON(w, http.StatusOK, provider.Summary())
	}
}

// NewSystemsHandler returns a handler for GET /analytics/systems.
func NewSystemsHandler(provider AnalyticsProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}
		respondJSON(w, http.StatusOK, provider.Systems())
	}
}

// NewAlertsHandler returns a handler for GET /analytics/alerts.
func NewAlertsHandler(provider AnalyticsProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}
		respondJSON(w, http.StatusOK, provider.Alerts())
	}
}

func respondJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
