package handlers

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/stuttgart-things/homerun2-scout/internal/digest"
)

const (
	defaultMessagesLimit = 100
	maxMessagesLimit     = 500
)

// MessageLister returns the messages of a window ending now.
type MessageLister interface {
	Messages(ctx context.Context, s digest.Schedule, f digest.MessageFilter) (digest.MessagesReport, error)
}

// NewMessagesHandler returns a handler for GET /analytics/messages: the
// messages of the hourly, daily or weekly window ending now, for consumers
// such as homerun2-ai-pitcher that need more than counts (#90).
//
// Query: schedule=hourly|daily|weekly (default daily), systems, severity and
// tags as comma-separated lists, limit (default 100, at most 500; the newest
// are kept), fields=message for the message text.
func NewMessagesHandler(lister MessageLister) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}

		schedule, ok := previewSchedule(w, r)
		if !ok {
			return
		}

		q := r.URL.Query()
		filter := digest.MessageFilter{
			Systems:    splitParam(q.Get("systems")),
			Severities: splitParam(q.Get("severity")),
			Tags:       splitParam(q.Get("tags")),
			Limit:      defaultMessagesLimit,
		}
		if v := q.Get("limit"); v != "" {
			n, err := strconv.Atoi(v)
			if err != nil || n < 1 || n > maxMessagesLimit {
				respondJSON(w, http.StatusBadRequest, map[string]string{"error": "limit must be 1 to " + strconv.Itoa(maxMessagesLimit) + ", not " + v})
				return
			}
			filter.Limit = n
		}
		for _, f := range splitParam(q.Get("fields")) {
			if f != "message" {
				respondJSON(w, http.StatusBadRequest, map[string]string{"error": "fields can only be message, not " + f})
				return
			}
			filter.Body = true
		}

		report, err := lister.Messages(r.Context(), schedule, filter)
		if err != nil {
			respondJSON(w, http.StatusServiceUnavailable, map[string]string{"error": err.Error()})
			return
		}
		respondJSON(w, http.StatusOK, report)
	}
}

// splitParam splits a comma-separated query value, dropping empty items.
func splitParam(v string) []string {
	var items []string
	for item := range strings.SplitSeq(v, ",") {
		if item = strings.TrimSpace(item); item != "" {
			items = append(items, item)
		}
	}
	return items
}
