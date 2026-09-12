package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stuttgart-things/homerun2-scout/internal/digest"
)

type fakePreviewer struct {
	got digest.Schedule
	err error
}

func (f *fakePreviewer) Preview(_ context.Context, s digest.Schedule) (digest.Report, error) {
	f.got = s
	return digest.Report{Schedule: s.Name, Message: digest.Message{Title: s.Label + ": 1 msgs 0 err 0 crit"}}, f.err
}

func TestDigestHandler(t *testing.T) {
	cases := []struct {
		query, schedule string
		status          int
	}{
		{"", "daily", http.StatusOK},
		{"?schedule=daily", "daily", http.StatusOK},
		{"?schedule=hourly", "hourly", http.StatusOK},
		{"?schedule=weekly", "", http.StatusBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.query, func(t *testing.T) {
			p := &fakePreviewer{}
			rr := httptest.NewRecorder()
			NewDigestHandler(p)(rr, httptest.NewRequest(http.MethodGet, "/analytics/digest"+tc.query, nil))
			if rr.Code != tc.status {
				t.Fatalf("status %d, want %d: %s", rr.Code, tc.status, rr.Body)
			}
			if tc.status != http.StatusOK {
				return
			}
			var report digest.Report
			if err := json.Unmarshal(rr.Body.Bytes(), &report); err != nil || report.Schedule != tc.schedule || p.got.Name != tc.schedule {
				t.Errorf("report %+v err %v previewed %q", report, err, p.got.Name)
			}
		})
	}

	failing := &fakePreviewer{err: errors.New("redisearch index has no NUMERIC timestamp_unix")}
	rr := httptest.NewRecorder()
	NewDigestHandler(failing)(rr, httptest.NewRequest(http.MethodGet, "/analytics/digest", nil))
	if rr.Code != http.StatusServiceUnavailable || !strings.Contains(rr.Body.String(), "NUMERIC") {
		t.Errorf("failing preview: %d %s", rr.Code, rr.Body)
	}

	rr = httptest.NewRecorder()
	NewDigestHandler(&fakePreviewer{})(rr, httptest.NewRequest(http.MethodPost, "/analytics/digest", nil))
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST: %d", rr.Code)
	}
}
