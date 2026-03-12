package handlers

import (
	"encoding/json"
	"net/http"
)

// NewRootHandler returns a handler for GET / that shows service info and available endpoints.
func NewRootHandler(version string) http.HandlerFunc {
	type endpoint struct {
		Path        string `json:"path"`
		Description string `json:"description"`
		Auth        bool   `json:"auth"`
	}

	resp := struct {
		Service   string     `json:"service"`
		Version   string     `json:"version"`
		Endpoints []endpoint `json:"endpoints"`
	}{
		Service: "homerun2-scout",
		Version: version,
		Endpoints: []endpoint{
			{Path: "/health", Description: "Health check and uptime", Auth: false},
			{Path: "/metrics", Description: "Prometheus metrics", Auth: false},
			{Path: "/analytics/summary", Description: "Aggregated event summary", Auth: true},
			{Path: "/analytics/systems", Description: "Events grouped by system", Auth: true},
			{Path: "/analytics/alerts", Description: "Alert counts and top systems", Auth: true},
		},
	}

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}
}
