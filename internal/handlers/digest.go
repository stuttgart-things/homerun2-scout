package handlers

import (
	"context"
	"net/http"

	"github.com/stuttgart-things/homerun2-scout/internal/digest"
)

// DigestPreviewer builds a digest without pitching it.
type DigestPreviewer interface {
	Preview(ctx context.Context, s digest.Schedule) (digest.Report, error)
}

// NewDigestHandler returns a handler for GET /analytics/digest?schedule=hourly|daily:
// the digest of the window ending now, as it would be pitched. It works whether
// or not the periodic digest is enabled.
func NewDigestHandler(previewer DigestPreviewer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}

		var schedule digest.Schedule
		switch name := r.URL.Query().Get("schedule"); name {
		case "", "daily":
			schedule, _ = digest.Daily("00:00")
		case "hourly":
			schedule = digest.Hourly()
		default:
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "schedule must be hourly or daily, not " + name})
			return
		}

		report, err := previewer.Preview(r.Context(), schedule)
		if err != nil {
			respondJSON(w, http.StatusServiceUnavailable, map[string]string{"error": err.Error()})
			return
		}
		respondJSON(w, http.StatusOK, report)
	}
}
