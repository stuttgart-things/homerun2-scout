package digest

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestSummarize(t *testing.T) {
	start, end := time.Unix(1000, 0), time.Unix(4600, 0)
	rows := []Row{
		{System: "kubernetes", Severity: "SUCCESS", Count: 700},
		{System: "kubernetes", Severity: "INFO", Count: 79},
		{System: "github", Severity: "ERROR", Count: 2},
		{System: "github", Severity: "error", Count: 1},
		{System: "github", Severity: "INFO", Count: 20},
		{System: "tabletennis", Severity: "CRITICAL", Count: 1},
		{System: "scout-digest", Severity: "success", Count: 24}, // its own messages
		{System: "homerun2-scout", Severity: "WARNING", Count: 3},
		{System: "", Severity: "INFO", Count: 5},
	}

	sum := Summarize(rows, start, end, []string{"scout-digest", "homerun2-scout"}, 2)

	if sum.Total != 803 {
		t.Errorf("Total = %d, want 803 (excluded systems and blank ones left out)", sum.Total)
	}
	wantSev := map[string]int64{"success": 700, "info": 99, "error": 3, "critical": 1}
	if !reflect.DeepEqual(sum.Severities, wantSev) {
		t.Errorf("Severities = %v, want %v", sum.Severities, wantSev)
	}
	if sum.Errors() != 3 || sum.Critical() != 1 {
		t.Errorf("Errors %d Critical %d", sum.Errors(), sum.Critical())
	}
	// Alerts first, then volume: github (3 alerts) before tabletennis (1) before kubernetes (0).
	wantTop := []SystemCount{{System: "github", Count: 23, Alerts: 3}, {System: "tabletennis", Count: 1, Alerts: 1}}
	if !reflect.DeepEqual(sum.TopSystems, wantTop) {
		t.Errorf("TopSystems = %+v, want %+v", sum.TopSystems, wantTop)
	}
	if !sum.Start.Equal(start) || !sum.End.Equal(end) {
		t.Errorf("window %v - %v", sum.Start, sum.End)
	}
}

func TestSummarizeEmptyAndTies(t *testing.T) {
	empty := Summarize(nil, time.Time{}, time.Time{}, nil, 3)
	if empty.Total != 0 || empty.Severities == nil || empty.TopSystems == nil {
		t.Errorf("empty summary = %+v: maps and lists must encode as {} and [], not null", empty)
	}

	tied := Summarize([]Row{
		{System: "b", Severity: "info", Count: 5},
		{System: "a", Severity: "info", Count: 5},
		{System: "c", Severity: "info", Count: 9},
	}, time.Time{}, time.Time{}, nil, 0)
	if len(tied.TopSystems) != 0 {
		t.Errorf("top 0 = %v", tied.TopSystems)
	}
	all := Summarize([]Row{
		{System: "b", Severity: "info", Count: 5},
		{System: "a", Severity: "info", Count: 5},
		{System: "c", Severity: "info", Count: 9},
	}, time.Time{}, time.Time{}, nil, 10)
	var names []string
	for _, sc := range all.TopSystems {
		names = append(names, sc.System)
	}
	if strings.Join(names, ",") != "c,a,b" {
		t.Errorf("order = %v, want c,a,b", names)
	}
}
