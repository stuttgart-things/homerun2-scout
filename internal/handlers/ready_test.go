package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stuttgart-things/homerun2-scout/internal/models"
)

type fakeReadiness struct {
	ready bool
	resp  models.ReadinessResponse
}

func (f fakeReadiness) Readiness() (bool, models.ReadinessResponse) { return f.ready, f.resp }

func TestReadyHandler(t *testing.T) {
	cases := []struct {
		name  string
		ready bool
		want  int
	}{
		{"ready", true, http.StatusOK},
		{"not ready", false, http.StatusServiceUnavailable},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := NewReadyHandler(fakeReadiness{ready: tc.ready, resp: models.ReadinessResponse{Status: "x", Reason: "because"}})
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/ready", nil))
			if rr.Code != tc.want {
				t.Fatalf("status = %d, want %d", rr.Code, tc.want)
			}
			var resp models.ReadinessResponse
			if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if resp.Reason != "because" {
				t.Errorf("body not passed through: %+v", resp)
			}
		})
	}
}

func TestReadyHandler_MethodNotAllowed(t *testing.T) {
	h := NewReadyHandler(fakeReadiness{ready: true})
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/ready", nil))
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusMethodNotAllowed)
	}
}
