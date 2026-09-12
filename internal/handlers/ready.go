package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/stuttgart-things/homerun2-scout/internal/models"
)

// ReadinessProvider reports whether scout can currently aggregate.
type ReadinessProvider interface {
	Readiness() (bool, models.ReadinessResponse)
}

// NewReadyHandler returns a handler for GET /ready: 200 while the RediSearch
// index exists and aggregation has succeeded recently, 503 otherwise, with the
// reason in the body. /health stays a plain liveness check, so a Redis outage
// takes scout out of its Service without restarting it (#75).
func NewReadyHandler(provider ReadinessProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}

		ready, resp := provider.Readiness()
		w.Header().Set("Content-Type", "application/json")
		if !ready {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		_ = json.NewEncoder(w).Encode(resp)
	}
}
