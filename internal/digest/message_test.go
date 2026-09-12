package digest

import (
	"strings"
	"testing"
	"time"
)

func summary(total, errors, critical int64) Summary {
	sev := map[string]int64{"info": total - errors - critical}
	if errors > 0 {
		sev["error"] = errors
	}
	if critical > 0 {
		sev["critical"] = critical
	}
	return Summary{Total: total, Severities: sev, TopSystems: []SystemCount{}}
}

func TestFormatSeverity(t *testing.T) {
	daily, _ := Daily("07:00")
	cases := []struct {
		name     string
		cur      Summary
		previous *Summary
		want     string
	}{
		{"critical", summary(10, 0, 1), nil, "error"},
		{"critical wins over fewer errors", summary(10, 1, 1), ptr(summary(10, 5, 0)), "error"},
		{"more errors than before", summary(10, 3, 0), ptr(summary(10, 2, 0)), "warning"},
		{"errors, not more than before", summary(10, 2, 0), ptr(summary(10, 2, 0)), "info"},
		{"errors, nothing to compare", summary(10, 2, 0), nil, "info"},
		{"quiet", summary(10, 0, 0), ptr(summary(10, 4, 0)), "success"},
		{"empty window", summary(0, 0, 0), nil, "success"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Format(daily, tc.cur, tc.previous, "scout-digest", time.UTC).Severity; got != tc.want {
				t.Errorf("severity = %q, want %q", got, tc.want)
			}
		})
	}
}

func ptr(s Summary) *Summary { return &s }

func TestFormatMessage(t *testing.T) {
	loc := berlin(t)
	daily, _ := Daily("07:00")
	end := time.Date(2026, 9, 12, 7, 0, 0, 0, loc)
	cur := Summary{
		Start: daily.Start(end), End: end, Total: 803,
		Severities: map[string]int64{"success": 700, "info": 99, "error": 3, "critical": 1, "custom": 0},
		TopSystems: []SystemCount{{System: "github", Count: 23, Alerts: 3}, {System: "tabletennis", Count: 1, Alerts: 1}},
	}
	prev := summary(760, 1, 0)

	msg := Format(daily, cur, &prev, "scout-digest", loc)

	if msg.Title != "24h: 803 msgs 3 err 1 crit" {
		t.Errorf("title = %q", msg.Title)
	}
	for _, r := range msg.Title {
		if r > 127 {
			t.Errorf("title %q has non-ASCII %q: led-catcher's BDF font cannot render it", msg.Title, r)
		}
	}
	if msg.System != "scout-digest" || msg.Author != Author || msg.Tags != "digest,daily" || msg.Timestamp != "2026-09-12T05:00:00Z" {
		t.Errorf("envelope = %+v", msg)
	}
	for _, want := range []string{
		"daily digest 2026-09-11 07:00 - 2026-09-12 07:00 (Europe/Berlin)",
		"Messages: 803 (previous 760, +43)",
		"Errors: 3 (previous 1, +2), critical: 1 (previous 0, +1)",
		"Severities: critical 1, error 3, success 700, info 99, custom 0",
		"Top systems: github 23 (3 alerts), tabletennis 1 (1 alerts)",
	} {
		if !strings.Contains(msg.Message, want) {
			t.Errorf("message lacks %q:\n%s", want, msg.Message)
		}
	}

	noPrev := Format(Hourly(), cur, nil, "scout-digest", loc)
	if !strings.HasPrefix(noPrev.Title, "1h: ") || !strings.Contains(noPrev.Message, "Messages: 803\n") || noPrev.Tags != "digest,hourly" {
		t.Errorf("without previous window: %+v", noPrev)
	}
}
