package digest

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

// Author is who a digest message says pitched it.
const Author = "homerun2-scout"

// severityOrder is the order the message lists severities in: the worst first.
var severityOrder = []string{"critical", "error", "warning", "success", "info", "debug"}

// Message is a digest as pitched to omni-pitcher's /pitch.
type Message struct {
	Title     string `json:"title"`
	Message   string `json:"message"`
	Severity  string `json:"severity"`
	Author    string `json:"author"`
	System    string `json:"system"`
	Tags      string `json:"tags"`
	Timestamp string `json:"timestamp"`
}

// Format builds the digest message for cur. previous is the window before,
// nil when it is not known - its messages may already have been deleted by
// retention.
//
// The title stays short and ASCII: led-catcher renders it on a 64x64 matrix
// with a BDF font. The severity says how the window went, so catchers can
// react to it like to any message:
//
//	critical messages in the window                  -> error
//	more error messages than in the window before    -> warning
//	error messages, but not more than before         -> info
//	neither                                          -> success
func Format(s Schedule, cur Summary, previous *Summary, system string, loc *time.Location) Message {
	msg := Message{
		Title:     fmt.Sprintf("%s: %d msgs %d err %d crit", s.Label, cur.Total, cur.Errors(), cur.Critical()),
		Author:    Author,
		System:    system,
		Tags:      "digest," + s.Name,
		Timestamp: cur.End.UTC().Format(time.RFC3339),
	}

	switch {
	case cur.Critical() > 0:
		msg.Severity = "error"
	case previous != nil && cur.Errors() > previous.Errors():
		msg.Severity = "warning"
	case cur.Errors() > 0:
		msg.Severity = "info"
	default:
		msg.Severity = "success"
	}

	lines := []string{
		fmt.Sprintf("%s digest %s - %s (%s)", s.Name,
			cur.Start.In(loc).Format("2006-01-02 15:04"), cur.End.In(loc).Format("2006-01-02 15:04"), loc),
		"Messages: " + withChange(cur.Total, previous, func(p *Summary) int64 { return p.Total }),
		"Errors: " + withChange(cur.Errors(), previous, (*Summary).errors) +
			", critical: " + withChange(cur.Critical(), previous, (*Summary).critical),
	}
	if sev := severities(cur); sev != "" {
		lines = append(lines, "Severities: "+sev)
	}
	if len(cur.TopSystems) > 0 {
		var top []string
		for _, sc := range cur.TopSystems {
			top = append(top, fmt.Sprintf("%s %d (%d alerts)", sc.System, sc.Count, sc.Alerts))
		}
		lines = append(lines, "Top systems: "+strings.Join(top, ", "))
	}
	msg.Message = strings.Join(lines, "\n")
	return msg
}

func (s *Summary) errors() int64   { return s.Errors() }
func (s *Summary) critical() int64 { return s.Critical() }

// withChange renders n, and its difference to the previous window when known.
func withChange(n int64, previous *Summary, of func(*Summary) int64) string {
	if previous == nil {
		return fmt.Sprint(n)
	}
	return fmt.Sprintf("%d (previous %d, %+d)", n, of(previous), n-of(previous))
}

// severities lists the window's severity counts, the worst first.
func severities(s Summary) string {
	names := make([]string, 0, len(s.Severities))
	for name := range s.Severities {
		names = append(names, name)
	}
	slices.SortFunc(names, func(a, b string) int {
		ia, ib := slices.Index(severityOrder, a), slices.Index(severityOrder, b)
		switch {
		case ia >= 0 && ib >= 0:
			return ia - ib
		case ia >= 0:
			return -1
		case ib >= 0:
			return 1
		default:
			return strings.Compare(a, b)
		}
	})
	parts := make([]string, 0, len(names))
	for _, name := range names {
		parts = append(parts, fmt.Sprintf("%s %d", name, s.Severities[name]))
	}
	return strings.Join(parts, ", ")
}
