package digest

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/stuttgart-things/homerun2-scout/internal/aggregator"
)

// maxScan is how many messages of a window are read at most, newest first.
// A window with more reports Truncated: its oldest messages were not read.
const maxScan = 10000

// maxBody caps the message text of an entry, in characters. k8s-pitcher sends
// whole objects as text.
const maxBody = 1000

// Entry is one message of a window, as pitched.
type Entry struct {
	Timestamp time.Time `json:"timestamp"`
	System    string    `json:"system"`
	Severity  string    `json:"severity"`
	Title     string    `json:"title"`
	Author    string    `json:"author,omitempty"`
	Tags      []string  `json:"tags,omitempty"`
	URL       string    `json:"url,omitempty"`
	// Message is the message text, only when asked for.
	Message string `json:"message,omitempty"`
}

// MessageFilter selects the messages of a window. An empty list matches all.
type MessageFilter struct {
	Systems []string
	// Severities match regardless of case.
	Severities []string
	// Tags: a message matches when it carries any of them.
	Tags []string
	// Limit is how many of the newest matching messages are kept.
	Limit int
	// Body keeps the message text, cut at maxBody characters.
	Body bool
}

// MessagesReport is what /analytics/messages answers: the matching messages of
// a window, oldest first.
type MessagesReport struct {
	Schedule string    `json:"schedule"`
	Start    time.Time `json:"start"`
	End      time.Time `json:"end"`
	// Total counts the matching messages, Omitted the oldest of them that
	// Limit left out.
	Total   int `json:"total"`
	Omitted int `json:"omitted"`
	// Truncated reports that the window held more than maxScan messages: the
	// oldest were not read, and Total is too low.
	Truncated bool `json:"truncated,omitempty"`
	// WindowIncomplete reports that retention has already deleted the oldest
	// part of the window.
	WindowIncomplete bool    `json:"windowIncomplete,omitempty"`
	Messages         []Entry `json:"messages"`
}

// Messages returns the messages of s's window ending now that match f. The
// digest's own system and the excluded systems are left out, as in the
// digest's counts.
func (r *Runner) Messages(ctx context.Context, s Schedule, f MessageFilter) (MessagesReport, error) {
	end := r.now()
	start := s.Start(end)
	entries, truncated, err := r.querier.Messages(ctx, start, end, maxScan)
	if err != nil {
		return MessagesReport{}, err
	}
	exclude := append([]string{r.cfg.System}, r.cfg.Exclude...)
	kept, total := SelectMessages(entries, f, exclude)
	return MessagesReport{
		Schedule: s.Name, Start: start, End: end,
		Total: total, Omitted: total - len(kept),
		Truncated: truncated, WindowIncomplete: r.beyondRetention(start),
		Messages: kept,
	}, nil
}

// SelectMessages filters entries, newest first, by f and exclude. It returns
// the newest f.Limit matches oldest first, and how many matched.
func SelectMessages(entries []Entry, f MessageFilter, exclude []string) ([]Entry, int) {
	kept := []Entry{}
	total := 0
	for _, e := range entries {
		if !matches(e, f, exclude) {
			continue
		}
		total++
		if len(kept) >= f.Limit {
			continue
		}
		e.Severity = strings.ToLower(e.Severity)
		if !f.Body {
			e.Message = ""
		} else if r := []rune(e.Message); len(r) > maxBody {
			e.Message = string(r[:maxBody]) + "..."
		}
		kept = append(kept, e)
	}
	slices.Reverse(kept)
	return kept, total
}

func matches(e Entry, f MessageFilter, exclude []string) bool {
	if e.System == "" || slices.Contains(exclude, e.System) {
		return false
	}
	if len(f.Systems) > 0 && !slices.Contains(f.Systems, e.System) {
		return false
	}
	if len(f.Severities) > 0 && !slices.ContainsFunc(f.Severities, func(s string) bool {
		return strings.EqualFold(s, e.Severity)
	}) {
		return false
	}
	if len(f.Tags) > 0 && !slices.ContainsFunc(f.Tags, func(t string) bool {
		return slices.Contains(e.Tags, t)
	}) {
		return false
	}
	return true
}

// searchBatch is how many documents one FT.SEARCH of Messages returns.
const searchBatch = 500

// Messages implements Querier with FT.SEARCH over the time range, newest
// first. Filtering happens in Go: system and severity are TEXT fields, whose
// query syntax would split names like homerun2-git-pitcher at the hyphen.
func (q RedisQuerier) Messages(ctx context.Context, start, end time.Time, max int) ([]Entry, bool, error) {
	info, err := q.Client.Do(ctx, "FT.INFO", q.Index).Result()
	if err != nil {
		return nil, false, fmt.Errorf("FT.INFO %s: %w", q.Index, err)
	}
	if !aggregator.HasNumericAttribute(info, aggregator.TimestampUnixField) {
		return nil, false, ErrNoNumericTimestamp
	}

	query := fmt.Sprintf("@%s:[%d (%d]", aggregator.TimestampUnixField, start.Unix(), end.Unix())
	var entries []Entry
	for offset := 0; offset < max; offset += searchBatch {
		result, err := q.Client.Do(ctx,
			"FT.SEARCH", q.Index, query,
			"SORTBY", aggregator.TimestampUnixField, "DESC",
			"LIMIT", strconv.Itoa(offset), strconv.Itoa(min(searchBatch, max-offset)),
			"TIMEOUT", "30000",
		).Result()
		if err != nil {
			return nil, false, fmt.Errorf("FT.SEARCH %s: %w", q.Index, err)
		}
		total, batch := parseSearchDocs(result)
		entries = append(entries, batch...)
		if offset+searchBatch >= total {
			return entries, false, nil
		}
	}
	return entries, true, nil
}

// searchDoc is the part of a homerun message Messages reads.
type searchDoc struct {
	Title         string `json:"title"`
	Message       string `json:"message"`
	Severity      string `json:"severity"`
	Author        string `json:"author"`
	Timestamp     string `json:"timestamp"`
	TimestampUnix int64  `json:"timestamp_unix"`
	System        string `json:"system"`
	Tags          string `json:"tags"`
	URL           string `json:"url"`
}

// parseSearchDocs reads a RESP2 FT.SEARCH reply on a JSON index:
// [total, key, ["$", json], key, ["$", json], ...]. Documents that do not
// parse are skipped.
func parseSearchDocs(result any) (int, []Entry) {
	arr, ok := result.([]any)
	if !ok || len(arr) == 0 {
		return 0, nil
	}
	total, _ := arr[0].(int64)
	var entries []Entry
	for i := 2; i < len(arr); i += 2 {
		fields, ok := arr[i].([]any)
		if !ok {
			continue
		}
		for j := 0; j+1 < len(fields); j += 2 {
			if name, _ := fields[j].(string); name != "$" {
				continue
			}
			raw, _ := fields[j+1].(string)
			var d searchDoc
			if err := json.Unmarshal([]byte(raw), &d); err != nil {
				continue
			}
			entries = append(entries, d.entry())
		}
	}
	return int(total), entries
}

func (d searchDoc) entry() Entry {
	ts := time.Unix(d.TimestampUnix, 0).UTC()
	if d.TimestampUnix == 0 {
		ts, _ = time.Parse(time.RFC3339, d.Timestamp)
	}
	var tags []string
	for t := range strings.SplitSeq(d.Tags, ",") {
		if t = strings.TrimSpace(t); t != "" {
			tags = append(tags, t)
		}
	}
	return Entry{
		Timestamp: ts, System: d.System, Severity: d.Severity, Title: d.Title,
		Author: d.Author, Tags: tags, URL: d.URL, Message: d.Message,
	}
}
