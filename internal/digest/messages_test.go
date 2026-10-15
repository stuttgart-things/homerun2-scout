package digest

import (
	"context"
	"strings"
	"testing"
	"time"
)

func entry(min int, system, severity, title string, tags ...string) Entry {
	return Entry{
		Timestamp: time.Date(2026, 9, 12, 7, min, 0, 0, time.UTC),
		System:    system, Severity: severity, Title: title, Tags: tags,
		Message: "text of " + title,
	}
}

func titles(entries []Entry) string {
	var t []string
	for _, e := range entries {
		t = append(t, e.Title)
	}
	return strings.Join(t, ",")
}

func TestSelectMessages(t *testing.T) {
	// Newest first, as the querier returns them.
	entries := []Entry{
		entry(50, "homerun2-git-pitcher", "SUCCESS", "release", "github", "release", "o/a"),
		entry(40, "ai-digest", "info", "digest"),
		entry(30, "homerun2-git-pitcher", "success", "merged", "github", "pull_request", "o/a"),
		entry(20, "kubernetes", "ERROR", "pod down", "k8s"),
		entry(10, "homerun2-git-pitcher", "info", "opened", "github", "pull_request", "o/b"),
		entry(5, "", "info", "no system"),
	}
	exclude := []string{"scout-digest", "ai-digest"}

	cases := []struct {
		name   string
		filter MessageFilter
		want   string
		total  int
	}{
		{"all, oldest first, excluded systems left out", MessageFilter{Limit: 10}, "opened,pod down,merged,release", 4},
		{"limit keeps the newest", MessageFilter{Limit: 2}, "merged,release", 4},
		{"systems", MessageFilter{Systems: []string{"kubernetes"}, Limit: 10}, "pod down", 1},
		{"severity ignores case", MessageFilter{Severities: []string{"success"}, Limit: 10}, "merged,release", 2},
		{"any of the tags", MessageFilter{Tags: []string{"release", "k8s"}, Limit: 10}, "pod down,release", 2},
		{"tags match whole tags", MessageFilter{Tags: []string{"pull"}, Limit: 10}, "", 0},
		{"excluded even when asked for", MessageFilter{Systems: []string{"ai-digest"}, Limit: 10}, "", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			kept, total := SelectMessages(entries, tc.filter, exclude)
			if got := titles(kept); got != tc.want || total != tc.total {
				t.Errorf("got %q (%d), want %q (%d)", got, total, tc.want, tc.total)
			}
			for _, e := range kept {
				if e.Message != "" {
					t.Errorf("%s: message text without Body", e.Title)
				}
				if e.Severity != strings.ToLower(e.Severity) {
					t.Errorf("%s: severity %q not lower-cased", e.Title, e.Severity)
				}
			}
		})
	}
}

func TestSelectMessagesBody(t *testing.T) {
	long := entry(1, "s", "info", "long")
	long.Message = strings.Repeat("ä", maxBody+5)
	kept, _ := SelectMessages([]Entry{long, entry(0, "s", "info", "short")}, MessageFilter{Limit: 10, Body: true}, nil)
	if kept[0].Message != "text of short" {
		t.Errorf("short body = %q", kept[0].Message)
	}
	if got := []rune(kept[1].Message); len(got) != maxBody+3 || !strings.HasSuffix(string(got), "...") {
		t.Errorf("long body has %d runes", len(got))
	}
}

func TestRunnerMessages(t *testing.T) {
	loc := berlin(t)
	now := time.Date(2026, 9, 12, 7, 0, 0, 0, loc)
	q := &fakeQuerier{
		entries: []Entry{
			entry(2, "scout-digest", "info", "own digest"),
			entry(1, "homerun2-git-pitcher", "success", "merged"),
		},
		truncated: true,
	}
	r := newTestRunner(t, &now, q, &fakeStore{}, &fakePitcher{})

	daily, _ := Daily("00:00")
	rep, err := r.Messages(context.Background(), daily, MessageFilter{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Schedule != "daily" || !rep.End.Equal(now) || !rep.Start.Equal(now.AddDate(0, 0, -1)) {
		t.Errorf("window %s %v - %v", rep.Schedule, rep.Start, rep.End)
	}
	if titles(rep.Messages) != "merged" || rep.Total != 1 || rep.Omitted != 0 || !rep.Truncated {
		t.Errorf("report %+v", rep)
	}
	if rep.WindowIncomplete {
		t.Error("daily window incomplete with 48h retention")
	}

	weekly, _ := Weekly("00:00")
	rep, err = r.Messages(context.Background(), weekly, MessageFilter{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if !rep.WindowIncomplete {
		t.Error("weekly window with 48h retention not marked incomplete")
	}
	if got := q.calls[len(q.calls)-1]; !got[0].Equal(now.AddDate(0, 0, -7)) || !got[1].Equal(now) {
		t.Errorf("weekly queried %v", got)
	}
}

func TestRunnerReportWindowIncomplete(t *testing.T) {
	loc := berlin(t)
	now := time.Date(2026, 9, 12, 7, 0, 0, 0, loc)
	r := newTestRunner(t, &now, &fakeQuerier{}, &fakeStore{}, &fakePitcher{})
	weekly, _ := Weekly("00:00")
	rep, err := r.Preview(context.Background(), weekly)
	if err != nil {
		t.Fatal(err)
	}
	if !rep.WindowIncomplete || rep.Previous != nil {
		t.Errorf("weekly preview with 48h retention: incomplete %v, previous %v", rep.WindowIncomplete, rep.Previous)
	}

	r.cfg.Retention = 0
	if rep, _ := r.Preview(context.Background(), weekly); rep.WindowIncomplete || rep.Previous == nil {
		t.Errorf("retention off: incomplete %v, previous %v", rep.WindowIncomplete, rep.Previous)
	}
}

func TestParseSearchDocs(t *testing.T) {
	reply := []any{
		int64(3),
		"messages:1", []any{"$", `{"title":"PR #1 merged","severity":"success","system":"homerun2-git-pitcher","author":"pat","tags":"github, pull_request,o/a,","url":"https://x/1","message":"body","timestamp":"2026-09-12T07:00:00Z","timestamp_unix":1789196400}`},
		"messages:2", []any{"$", `not json`},
		"messages:3", []any{"$", `{"title":"old","system":"s","timestamp":"2026-09-12T06:00:00Z"}`},
	}
	total, entries := parseSearchDocs(reply)
	if total != 3 || len(entries) != 2 {
		t.Fatalf("total %d, %d entries", total, len(entries))
	}
	e := entries[0]
	if e.Title != "PR #1 merged" || e.System != "homerun2-git-pitcher" || e.URL != "https://x/1" || e.Author != "pat" ||
		strings.Join(e.Tags, "|") != "github|pull_request|o/a" || e.Timestamp.Unix() != 1789196400 || e.Message != "body" {
		t.Errorf("entry %+v", e)
	}
	if want := time.Date(2026, 9, 12, 6, 0, 0, 0, time.UTC); !entries[1].Timestamp.Equal(want) {
		t.Errorf("RFC3339 fallback: %v", entries[1].Timestamp)
	}
	if total, entries := parseSearchDocs("unexpected"); total != 0 || entries != nil {
		t.Error("non-array reply parsed")
	}
}
