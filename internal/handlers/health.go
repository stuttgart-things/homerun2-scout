package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/stuttgart-things/homerun2-scout/internal/models"
)

// NewHealthHandler returns a handler for GET /health.
func NewHealthHandler(version, commit, date string, startTime time.Time) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}

		resp := models.HealthResponse{
			Status:  "ok",
			Version: version,
			Commit:  commit,
			Date:    date,
			Uptime:  time.Since(startTime).Truncate(time.Second).String(),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}
