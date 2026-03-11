package retention

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// Cleaner periodically prunes old entries from a RediSearch index based on TTL.
type Cleaner struct {
	client   *redis.Client
	index    string
	ttl      time.Duration
	interval time.Duration
	cancel   context.CancelFunc
	done     chan struct{}
}

// New creates a new retention Cleaner.
func New(client *redis.Client, index string, ttl, interval time.Duration) *Cleaner {
	return &Cleaner{
		client:   client,
		index:    index,
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
				c.cleanup(ctx)
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

func (c *Cleaner) cleanup(ctx context.Context) {
	cutoff := time.Now().Add(-c.ttl).Unix()
	slog.Debug("running retention cleanup", "cutoffUnix", cutoff)

	// FT.SEARCH <index> @timestamp:[-inf <cutoff>] NOCONTENT LIMIT 0 1000
	query := fmt.Sprintf("@timestamp:[-inf %d]", cutoff)
	args := []interface{}{
		"FT.SEARCH", c.index, query,
		"NOCONTENT",
		"LIMIT", "0", "1000",
	}

	result, err := c.client.Do(ctx, args...).Result()
	if err != nil {
		slog.Warn("retention cleanup search failed", "error", err)
		return
	}

	keys := parseSearchKeys(result)
	if len(keys) == 0 {
		slog.Debug("retention cleanup: no expired entries")
		return
	}

	// Delete expired keys
	deleted := 0
	for _, key := range keys {
		if err := c.client.Del(ctx, key).Err(); err != nil {
			slog.Warn("retention cleanup: failed to delete key", "key", key, "error", err)
			continue
		}
		deleted++
	}

	slog.Info("retention cleanup complete", "found", len(keys), "deleted", deleted)
}

// parseSearchKeys extracts document keys from FT.SEARCH NOCONTENT response.
// FT.SEARCH NOCONTENT returns: [totalResults, key1, key2, ...]
func parseSearchKeys(result interface{}) []string {
	arr, ok := result.([]interface{})
	if !ok || len(arr) < 2 {
		return nil
	}

	// First element is total count
	total, ok := arr[0].(int64)
	if !ok || total == 0 {
		// Try string conversion
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
