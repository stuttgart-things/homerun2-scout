package digest

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

type fakeQuerier struct {
	rows  []Row
	err   error
	calls [][2]time.Time
}

func (q *fakeQuerier) Window(_ context.Context, start, end time.Time) ([]Row, error) {
	q.calls = append(q.calls, [2]time.Time{start, end})
	return q.rows, q.err
}

// fakeStore is shared by several runners the way Redis is shared by replicas.
type fakeStore struct {
	mu   sync.Mutex
	keys map[string]bool
}

func (s *fakeStore) Claim(_ context.Context, key string, _ time.Duration) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.keys == nil {
		s.keys = map[string]bool{}
	}
	if s.keys[key] {
		return false, nil
	}
	s.keys[key] = true
	return true, nil
}

func (s *fakeStore) Release(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.keys, key)
	return nil
}

type fakePitcher struct {
	msgs []Message
	err  error
}

func (p *fakePitcher) Pitch(_ context.Context, msg Message) error {
	if p.err != nil {
		return p.err
	}
	p.msgs = append(p.msgs, msg)
	return nil
}

func newTestRunner(t *testing.T, now *time.Time, q Querier, store Store, p Pitcher) *Runner {
	t.Helper()
	daily, _ := Daily("07:00")
	r := NewRunner(Config{
		Schedules: []Schedule{Hourly(), daily}, Location: berlin(t),
		TopSystems: 3, System: "scout-digest", Retention: 48 * time.Hour,
	}, q, store, p)
	r.now = func() time.Time { return *now }
	return r
}

func TestRunnerPitchesEachWindowOnce(t *testing.T) {
	loc := berlin(t)
	now := time.Date(2026, 9, 12, 7, 1, 0, 0, loc)
	q := &fakeQuerier{rows: []Row{{System: "github", Severity: "ERROR", Count: 2}}}
	store := &fakeStore{}
	p := &fakePitcher{}
	r := newTestRunner(t, &now, q, store, p)

	r.RunOnce(context.Background())
	if len(p.msgs) != 2 {
		t.Fatalf("pitched %d digests at 07:01, want hourly and daily", len(p.msgs))
	}
	if p.msgs[0].Tags != "digest,hourly" || p.msgs[1].Tags != "digest,daily" {
		t.Errorf("tags = %q, %q", p.msgs[0].Tags, p.msgs[1].Tags)
	}

	// Later in the same hour, and a second replica sharing the store.
	now = now.Add(20 * time.Minute)
	r.RunOnce(context.Background())
	newTestRunner(t, &now, q, store, p).RunOnce(context.Background())
	if len(p.msgs) != 2 {
		t.Errorf("pitched %d digests, want no second copy of either window", len(p.msgs))
	}

	now = time.Date(2026, 9, 12, 8, 0, 30, 0, loc)
	r.RunOnce(context.Background())
	if len(p.msgs) != 3 || p.msgs[2].Tags != "digest,hourly" {
		t.Errorf("at 08:00 want the next hourly digest only, got %d", len(p.msgs))
	}
}

func TestRunnerSkipsWindowsTooLongAgo(t *testing.T) {
	loc := berlin(t)
	// 07:31: the hourly digest of 07:00 is 31 minutes late, the daily one not yet an hour.
	now := time.Date(2026, 9, 12, 7, 31, 0, 0, loc)
	p := &fakePitcher{}
	r := newTestRunner(t, &now, &fakeQuerier{}, &fakeStore{}, p)
	r.RunOnce(context.Background())
	if len(p.msgs) != 1 || p.msgs[0].Tags != "digest,daily" {
		t.Errorf("pitched %+v, want the daily digest only", p.msgs)
	}

	// A scout started in the afternoon does not pitch this morning's digest.
	now = time.Date(2026, 9, 12, 15, 45, 0, 0, loc)
	p2 := &fakePitcher{}
	newTestRunner(t, &now, &fakeQuerier{}, &fakeStore{}, p2).RunOnce(context.Background())
	if len(p2.msgs) != 0 {
		t.Errorf("pitched %+v at 15:45", p2.msgs)
	}
}

func TestRunnerRetriesAFailedPitch(t *testing.T) {
	loc := berlin(t)
	now := time.Date(2026, 9, 12, 9, 2, 0, 0, loc)
	store := &fakeStore{}
	failing := &fakePitcher{err: errors.New("omni-pitcher answered 503")}
	r := newTestRunner(t, &now, &fakeQuerier{}, store, failing)
	r.RunOnce(context.Background())
	if len(store.keys) != 0 {
		t.Errorf("a failed pitch must release its claim, keys %v", store.keys)
	}

	ok := &fakePitcher{}
	r.pitcher = ok
	now = now.Add(time.Minute)
	r.RunOnce(context.Background())
	if len(ok.msgs) != 1 {
		t.Errorf("retry pitched %d digests, want 1", len(ok.msgs))
	}

	failingQuery := &fakeQuerier{err: ErrNoNumericTimestamp}
	now = time.Date(2026, 9, 12, 10, 0, 0, 0, loc)
	r2 := newTestRunner(t, &now, failingQuery, store, ok)
	r2.RunOnce(context.Background())
	if len(ok.msgs) != 1 || store.keys["scout:digest:hourly:"+strconv.FormatInt(now.Unix(), 10)] {
		t.Errorf("a failed query must not pitch and must release, msgs %d keys %v", len(ok.msgs), store.keys)
	}
}

func TestBuildComparesOnlyWhatRetentionHolds(t *testing.T) {
	loc := berlin(t)
	daily, _ := Daily("07:00")
	end := time.Date(2026, 9, 12, 7, 0, 0, 0, loc)
	now := end.Add(time.Minute)

	// 48h retention at 07:01 has deleted the first minute of the day before -
	// within the daily lateness, so the days are compared.
	q := &fakeQuerier{}
	r := newTestRunner(t, &now, q, &fakeStore{}, &fakePitcher{})
	_, report, err := r.Build(context.Background(), daily, end)
	if err != nil {
		t.Fatal(err)
	}
	if report.Previous == nil || len(q.calls) != 2 || !q.calls[0][0].Equal(daily.Start(end)) || !q.calls[0][1].Equal(end) ||
		!q.calls[1][1].Equal(daily.Start(end)) {
		t.Errorf("daily with 48h retention: calls %v previous %v", q.calls, report.Previous)
	}

	// 36h retention holds only half of the day before: no comparison.
	q1 := &fakeQuerier{}
	r1 := newTestRunner(t, &now, q1, &fakeStore{}, &fakePitcher{})
	r1.cfg.Retention = 36 * time.Hour
	_, report, _ = r1.Build(context.Background(), daily, end)
	if report.Previous != nil || len(q1.calls) != 1 {
		t.Errorf("36h retention cannot hold the day before: calls %v previous %+v", q1.calls, report.Previous)
	}

	q2 := &fakeQuerier{}
	r2 := newTestRunner(t, &now, q2, &fakeStore{}, &fakePitcher{})
	r2.cfg.Retention = 72 * time.Hour
	_, report, _ = r2.Build(context.Background(), Hourly(), end)
	if report.Previous == nil || len(q2.calls) != 2 || !q2.calls[1][1].Equal(end.Add(-time.Hour)) {
		t.Errorf("hourly with 72h retention should compare the hour before: calls %v previous %v", q2.calls, report.Previous)
	}

	q3 := &fakeQuerier{rows: []Row{{System: "scout-digest", Severity: "success", Count: 9}, {System: "other", Severity: "info", Count: 1}}}
	r3 := newTestRunner(t, &now, q3, &fakeStore{}, &fakePitcher{})
	_, report, _ = r3.Build(context.Background(), Hourly(), end)
	if report.Current.Total != 1 {
		t.Errorf("the digest's own system must not count: total %d", report.Current.Total)
	}
}

func TestHTTPPitcher(t *testing.T) {
	var got Message
	var auth, path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth, path = r.Header.Get("Authorization"), r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&got)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	msg := Message{Title: "24h: 1 msgs 0 err 0 crit", System: "scout-digest"}
	if err := (HTTPPitcher{URL: srv.URL, Token: "t"}).Pitch(context.Background(), msg); err != nil {
		t.Fatal(err)
	}
	if path != "/pitch" || auth != "Bearer t" || got != msg {
		t.Errorf("path %q auth %q got %+v", path, auth, got)
	}

	failing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusUnauthorized) }))
	defer failing.Close()
	if err := (HTTPPitcher{URL: failing.URL}).Pitch(context.Background(), msg); err == nil || !strings.Contains(err.Error(), "401") {
		t.Errorf("err = %v, want the status", err)
	}
}

func TestParseRows(t *testing.T) {
	reply := []any{int64(2),
		[]any{"system", "github", "severity", "ERROR", "count", "3"},
		[]any{"system", "kubernetes", "severity", "SUCCESS", "count", "700"},
		"garbage",
	}
	rows := parseRows(reply)
	if len(rows) != 2 || rows[0] != (Row{System: "github", Severity: "ERROR", Count: 3}) || rows[1].Count != 700 {
		t.Errorf("rows = %+v", rows)
	}
	if parseRows("OK") != nil || parseRows([]any{}) != nil {
		t.Error("unexpected replies must give no rows")
	}
}
