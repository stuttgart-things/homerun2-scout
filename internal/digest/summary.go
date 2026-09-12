package digest

import (
	"cmp"
	"slices"
	"strings"
	"time"
)

// Row is one group of the window query: the messages of one system at one
// severity, as pitched.
type Row struct {
	System   string
	Severity string
	Count    int64
}

// SystemCount is one system's share of a window.
type SystemCount struct {
	System string `json:"system"`
	Count  int64  `json:"count"`
	// Alerts counts its error and critical messages.
	Alerts int64 `json:"alerts"`
}

// Summary is what one window held.
type Summary struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
	Total int64     `json:"total"`
	// Severities counts messages per lower-cased severity: pitchers send
	// ERROR as well as error, and both are the same to the catchers.
	Severities map[string]int64 `json:"severities"`
	// TopSystems are the systems with the most alerts, then the most messages.
	TopSystems []SystemCount `json:"topSystems"`
}

// Errors returns the window's error messages.
func (s Summary) Errors() int64 { return s.Severities["error"] }

// Critical returns the window's critical messages.
func (s Summary) Critical() int64 { return s.Severities["critical"] }

// Summarize folds the rows of a window into a Summary, leaving out the systems
// in exclude - the digest's own messages above all, which would otherwise count
// themselves in the next window - and keeping the top systems.
func Summarize(rows []Row, start, end time.Time, exclude []string, top int) Summary {
	sum := Summary{Start: start, End: end, Severities: map[string]int64{}, TopSystems: []SystemCount{}}
	systems := map[string]*SystemCount{}

	for _, r := range rows {
		if r.System == "" || slices.Contains(exclude, r.System) {
			continue
		}
		severity := strings.ToLower(r.Severity)
		sum.Total += r.Count
		sum.Severities[severity] += r.Count

		sc, ok := systems[r.System]
		if !ok {
			sc = &SystemCount{System: r.System}
			systems[r.System] = sc
		}
		sc.Count += r.Count
		if severity == "error" || severity == "critical" {
			sc.Alerts += r.Count
		}
	}

	for _, sc := range systems {
		sum.TopSystems = append(sum.TopSystems, *sc)
	}
	slices.SortFunc(sum.TopSystems, func(a, b SystemCount) int {
		return cmp.Or(cmp.Compare(b.Alerts, a.Alerts), cmp.Compare(b.Count, a.Count), strings.Compare(a.System, b.System))
	})
	if top >= 0 && len(sum.TopSystems) > top {
		sum.TopSystems = sum.TopSystems[:top]
	}
	return sum
}
