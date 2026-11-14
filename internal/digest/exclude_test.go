package digest

import (
	"testing"
	"time"
)

func TestExcluded(t *testing.T) {
	ex := []string{"scout-digest", "vm-*", "k8sgpt-*", "inventory-platform-sthings"}
	for system, want := range map[string]bool{
		"scout-digest":               true,
		"scout-digest-2":             false, // exact, as before
		"vm-sthings-app":             true,
		"vm-":                        true,
		"vmware":                     false,
		"k8sgpt-platform-sthings":    true,
		"inventory-platform-sthings": true,
		"inventory-other":            false,
		"flux":                       false,
	} {
		if got := Excluded(system, ex); got != want {
			t.Errorf("%s: %v, want %v", system, got, want)
		}
	}
	if !Excluded("anything", []string{"*"}) {
		t.Error(`"*" excludes everything`)
	}
	if Excluded("x", nil) {
		t.Error("nil list")
	}
}

func TestSummarizePrefix(t *testing.T) {
	rows := []Row{{"vm-a", "warning", 3}, {"vm-b", "error", 1}, {"flux", "info", 2}}
	s := Summarize(rows, time.Time{}, time.Time{}, []string{"vm-*"}, 3)
	if s.Total != 2 || len(s.TopSystems) != 1 || s.TopSystems[0].System != "flux" {
		t.Errorf("summary %+v", s)
	}
}

func TestSelectMessagesPrefix(t *testing.T) {
	entries := []Entry{{System: "endpoints-p"}, {System: "flux"}}
	kept, total := SelectMessages(entries, MessageFilter{Limit: 10}, []string{"endpoints-*"})
	if total != 1 || kept[0].System != "flux" {
		t.Errorf("kept %+v", kept)
	}
}
