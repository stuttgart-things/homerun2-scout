package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/stuttgart-things/homerun2-scout/internal/digest"
)

type fakeLister struct {
	schedule digest.Schedule
	filter   digest.MessageFilter
	err      error
}

func (f *fakeLister) Messages(_ context.Context, s digest.Schedule, filter digest.MessageFilter) (digest.MessagesReport, error) {
	f.schedule, f.filter = s, filter
	return digest.MessagesReport{Schedule: s.Name, Messages: []digest.Entry{{Title: "PR #1"}}}, f.err
}

func TestMessagesHandler(t *testing.T) {
	get := func(lister MessageLister, query string) *httptest.ResponseRecorder {
		rr := httptest.NewRecorder()
		NewMessagesHandler(lister)(rr, httptest.NewRequest(http.MethodGet, "/analytics/messages"+query, nil))
		return rr
	}

	l := &fakeLister{}
	rr := get(l, "")
	if rr.Code != http.StatusOK || l.schedule.Name != "daily" || l.filter.Limit != 100 || l.filter.Body ||
		l.filter.Systems != nil || l.filter.Severities != nil || l.filter.Tags != nil {
		t.Fatalf("defaults: %d %+v %+v", rr.Code, l.schedule, l.filter)
	}
	var rep digest.MessagesReport
	if err := json.Unmarshal(rr.Body.Bytes(), &rep); err != nil || rep.Schedule != "daily" || len(rep.Messages) != 1 {
		t.Errorf("body %s: %v", rr.Body, err)
	}

	l = &fakeLister{}
	rr = get(l, "?schedule=weekly&systems=homerun2-git-pitcher,%20k8s-platform&severity=error,critical&tags=release,,pull_request&limit=500&fields=message")
	if rr.Code != http.StatusOK || l.schedule.Name != "weekly" || l.filter.Limit != 500 || !l.filter.Body ||
		!slices.Equal(l.filter.Systems, []string{"homerun2-git-pitcher", "k8s-platform"}) ||
		!slices.Equal(l.filter.Severities, []string{"error", "critical"}) ||
		!slices.Equal(l.filter.Tags, []string{"release", "pull_request"}) {
		t.Errorf("filters: %d %+v", rr.Code, l.filter)
	}

	for _, q := range []string{"?schedule=monthly", "?limit=0", "?limit=501", "?limit=x", "?fields=title"} {
		if rr := get(&fakeLister{}, q); rr.Code != http.StatusBadRequest {
			t.Errorf("%s: %d, want 400", q, rr.Code)
		}
	}

	rr = get(&fakeLister{err: errors.New("redisearch index has no NUMERIC timestamp_unix")}, "")
	if rr.Code != http.StatusServiceUnavailable || !strings.Contains(rr.Body.String(), "NUMERIC") {
		t.Errorf("failing lister: %d %s", rr.Code, rr.Body)
	}

	rr = httptest.NewRecorder()
	NewMessagesHandler(&fakeLister{})(rr, httptest.NewRequest(http.MethodPost, "/analytics/messages", nil))
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST: %d", rr.Code)
	}
}
