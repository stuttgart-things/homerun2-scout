package retention

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/stuttgart-things/homerun2-scout/internal/aggregator"
)

// Cleaner periodically prunes old entries from a RediSearch index and Redis Stream based on TTL.
type Cleaner struct {
	client   *redis.Client
	index    string
	stream   string
	ttl      time.Duration
	interval time.Duration
	cancel   context.CancelFunc
	done     chan struct{}
}

// New creates a new retention Cleaner.
// The stream name defaults to the index name (both are typically "messages").
func New(client *redis.Client, index string, ttl, interval time.Duration) *Cleaner {
	return &Cleaner{
		client:   client,
		index:    index,
		stream:   index,
		ttl:      ttl,
		interval: interval,
		done:     make(chan struct{}),
	}
}

// Start begins the periodic cleanup loop.
func (c *Cleaner) Start(ctx context.Context) {
	ctx, c.cancel = context.WithCancel(ctx)

	go func() {
		defer close(c.done)
		ticker := time.NewTicker(c.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				c.cleanupDocuments(ctx)
				c.cleanupStream(ctx)
			}
		}
	}()

	slog.Info("retention cleaner started", "ttl", c.ttl, "interval", c.interval)
}

// Stop stops the cleanup loop.
func (c *Cleaner) Stop() {
	if c.cancel != nil {
		c.cancel()
		<-c.done
	}
	slog.Info("retention cleaner stopped")
}

// batchSize is how many documents one FT.SEARCH of the cleanup returns.
const batchSize = 500

// cleanupDocuments removes JSON documents from the RediSearch index that are older than the TTL.
//
// With a NUMERIC timestamp_unix in the index (homerun-library v4.5.0+ writes it)
// the expired documents are one range query away. Documents written before
// that carry only the RFC3339 string, which the index cannot compare: they are
// fetched and compared in Go, as every document was before. An index without
// the attribute would answer the range query with nothing - no error - so the
// attribute is checked first and its absence means the Go path for everything.
func (c *Cleaner) cleanupDocuments(ctx context.Context) {
	cutoff := time.Now().Add(-c.ttl)
	slog.Debug("running document retention cleanup", "cutoff", cutoff.Format(time.RFC3339))

	legacyQuery := "*"
	totalDeleted := 0
	if c.indexHasNumericTimestamp(ctx) {
		totalDeleted += c.cleanupByRange(ctx, cutoff)
		legacyQuery = withoutNumericTimestampQuery()
	}
	totalDeleted += c.cleanupByParsedTimestamp(ctx, legacyQuery, cutoff)

	if totalDeleted > 0 {
		slog.Info("document retention cleanup complete", "deleted", totalDeleted)
	} else {
		slog.Debug("document retention cleanup: no expired entries")
	}
}

// indexHasNumericTimestamp reports whether the index declares timestamp_unix
// NUMERIC. A failed FT.INFO counts as no, which keeps the Go path.
func (c *Cleaner) indexHasNumericTimestamp(ctx context.Context) bool {
	info, err := c.client.Do(ctx, "FT.INFO", c.index).Result()
	if err != nil {
		slog.Warn("retention: FT.INFO failed, comparing timestamps in Go", "error", err)
		return false
	}
	return aggregator.HasNumericAttribute(info, aggregator.TimestampUnixField)
}

// expiredQuery matches documents whose event time is before cutoff.
func expiredQuery(cutoff time.Time) string {
	return fmt.Sprintf("@%s:[-inf (%d]", aggregator.TimestampUnixField, cutoff.Unix())
}

// withoutNumericTimestampQuery matches documents that have no timestamp_unix -
// those written before homerun-library v4.5.0.
func withoutNumericTimestampQuery() string {
	return fmt.Sprintf("-@%s:[-inf +inf]", aggregator.TimestampUnixField)
}

// cleanupByRange deletes the documents expiredQuery matches, a batch at a time.
// Deleted documents leave the index, so every batch starts at offset 0; a batch
// that deletes nothing ends the loop rather than fetching it again.
func (c *Cleaner) cleanupByRange(ctx context.Context, cutoff time.Time) int {
	total := 0
	for {
		result, err := c.client.Do(ctx,
			"FT.SEARCH", c.index, expiredQuery(cutoff),
			"NOCONTENT",
			"LIMIT", "0", strconv.Itoa(batchSize),
			"TIMEOUT", "30000",
		).Result()
		if err != nil {
			slog.Warn("retention range search failed", "error", err)
			return total
		}

		keys := parseSearchKeys(result)
		deleted := 0
		for _, key := range keys {
			if err := c.client.Del(ctx, key).Err(); err != nil {
				slog.Warn("retention: failed to delete key", "key", key, "error", err)
				continue
			}
			deleted++
		}
		total += deleted

		if len(keys) < batchSize || deleted == 0 {
			return total
		}
	}
}

// cleanupByParsedTimestamp fetches the documents query matches with their RFC3339
// timestamp, paginated, and deletes those before cutoff.
func (c *Cleaner) cleanupByParsedTimestamp(ctx context.Context, query string, cutoff time.Time) int {
	offset := 0
	totalDeleted := 0

	for {
		args := []any{
			"FT.SEARCH", c.index, query,
			"RETURN", "1", "timestamp",
			"LIMIT", strconv.Itoa(offset), strconv.Itoa(batchSize),
			"TIMEOUT", "30000",
		}

		result, err := c.client.Do(ctx, args...).Result()
		if err != nil {
			slog.Warn("retention cleanup search failed", "error", err)
			return totalDeleted
		}

		entries := parseSearchEntries(result)
		if len(entries) == 0 {
			break
		}

		// Check each document's timestamp and delete if expired
		deleted := 0
		for _, entry := range entries {
			ts, err := time.Parse(time.RFC3339, entry.timestamp)
			if err != nil {
				// Try alternative format without timezone
				ts, err = time.Parse("2006-01-02T15:04:05Z", entry.timestamp)
				if err != nil {
					continue
				}
			}
			if ts.Before(cutoff) {
				if err := c.client.Del(ctx, entry.key).Err(); err != nil {
					slog.Warn("retention: failed to delete key", "key", entry.key, "error", err)
					continue
				}
				deleted++
			}
		}
		totalDeleted += deleted

		// If we got fewer results than batch size, we're done
		if len(entries) < batchSize {
			break
		}

		// Only advance offset by entries we didn't delete (deleted ones shift the index)
		offset += len(entries) - deleted
	}

	return totalDeleted
}

// cleanupStream trims Redis Stream entries older than the TTL using XTRIM MINID.
func (c *Cleaner) cleanupStream(ctx context.Context) {
	cutoff := time.Now().Add(-c.ttl)
	// Redis Stream IDs are millisecond timestamps
	minID := fmt.Sprintf("%d-0", cutoff.UnixMilli())

	slog.Debug("running stream retention cleanup", "stream", c.stream, "minID", minID)

	trimmed, err := c.client.XTrimMinID(ctx, c.stream, minID).Result()
	if err != nil {
		slog.Warn("stream retention trim failed", "stream", c.stream, "error", err)
		return
	}

	if trimmed > 0 {
		slog.Info("stream retention cleanup complete", "stream", c.stream, "trimmed", trimmed)
	} else {
		slog.Debug("stream retention cleanup: no entries to trim")
	}
}

// searchEntry holds a document key and its timestamp from FT.SEARCH results.
type searchEntry struct {
	key       string
	timestamp string
}

// parseSearchEntries extracts document keys and timestamps from FT.SEARCH RETURN 1 timestamp response.
// Response format: [totalResults, key1, [timestamp, value1], key2, [timestamp, value2], ...]
func parseSearchEntries(result any) []searchEntry {
	arr, ok := result.([]any)
	if !ok || len(arr) < 3 {
		return nil
	}

	// First element is total count
	total, ok := arr[0].(int64)
	if !ok || total == 0 {
		if s, ok := arr[0].(string); ok {
			t, err := strconv.ParseInt(s, 10, 64)
			if err != nil || t == 0 {
				return nil
			}
		} else {
			return nil
		}
	}

	var entries []searchEntry
	// Results come in pairs: key, [field, value]
	for i := 1; i+1 < len(arr); i += 2 {
		key, ok := arr[i].(string)
		if !ok {
			continue
		}
		fields, ok := arr[i+1].([]any)
		if !ok || len(fields) < 2 {
			continue
		}
		// fields = ["timestamp", "2026-03-11T00:30:06Z"]
		if ts, ok := fields[1].(string); ok {
			entries = append(entries, searchEntry{key: key, timestamp: ts})
		}
	}

	return entries
}

// parseSearchKeys extracts document keys from FT.SEARCH NOCONTENT response.
// FT.SEARCH NOCONTENT returns: [totalResults, key1, key2, ...]
// Kept for backward compatibility.
func parseSearchKeys(result any) []string {
	arr, ok := result.([]any)
	if !ok || len(arr) < 2 {
		return nil
	}

	total, ok := arr[0].(int64)
	if !ok || total == 0 {
		if s, ok := arr[0].(string); ok {
			t, err := strconv.ParseInt(s, 10, 64)
			if err != nil || t == 0 {
				return nil
			}
		} else {
			return nil
		}
	}

	var keys []string
	for i := 1; i < len(arr); i++ {
		if key, ok := arr[i].(string); ok {
			keys = append(keys, key)
		}
	}
	return keys
}
