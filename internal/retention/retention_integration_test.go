//go:build integration

package retention

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/stuttgart-things/homerun2-scout/internal/aggregator"
)

// TestCleanupDocuments runs the retention against a real redis-stack:
//
//	REDIS_ADDR=localhost REDIS_PORT=6379 go test -tags integration ./internal/retention/
//
// It covers both index generations: one declaring timestamp_unix NUMERIC, where
// expired documents are found by range and those written before
// homerun-library v4.5.0 by comparing their RFC3339 string in Go, and one
// created before the numeric field, where everything takes the Go path.
func TestCleanupDocuments(t *testing.T) {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		t.Skip("set REDIS_ADDR (and REDIS_PORT) to a redis-stack")
	}
	port := os.Getenv("REDIS_PORT")
	if port == "" {
		port = "6379"
	}
	client := redis.NewClient(&redis.Options{Addr: addr + ":" + port, Protocol: 2})
	defer func() { _ = client.Close() }()
	ctx := context.Background()

	now := time.Now()
	expired := now.Add(-72 * time.Hour)
	fresh := now.Add(-time.Hour)

	for _, numeric := range []bool{true, false} {
		t.Run(fmt.Sprintf("numeric=%v", numeric), func(t *testing.T) {
			run := fmt.Sprintf("ret%d:", now.UnixNano())
			index := strings.TrimSuffix(run, ":")
			schema := []any{"FT.CREATE", index, "ON", "JSON", "PREFIX", "1", run, "SCHEMA",
				"$.timestamp", "AS", "timestamp", "TEXT"}
			if numeric {
				schema = append(schema, "$."+aggregator.TimestampUnixField, "AS", aggregator.TimestampUnixField, "NUMERIC", "SORTABLE")
			}
			if err := client.Do(ctx, schema...).Err(); err != nil {
				t.Fatal(err)
			}
			defer func() { _ = client.Do(ctx, "FT.DROPINDEX", index).Err() }()

			docs := map[string]string{
				"new-expired":    fmt.Sprintf(`{"timestamp":%q,"timestamp_unix":%d}`, expired.UTC().Format(time.RFC3339), expired.Unix()),
				"new-fresh":      fmt.Sprintf(`{"timestamp":%q,"timestamp_unix":%d}`, fresh.UTC().Format(time.RFC3339), fresh.Unix()),
				"legacy-expired": fmt.Sprintf(`{"timestamp":%q}`, expired.UTC().Format(time.RFC3339)),
				"legacy-fresh":   fmt.Sprintf(`{"timestamp":%q}`, fresh.UTC().Format(time.RFC3339)),
			}
			for name, doc := range docs {
				if err := client.Do(ctx, "JSON.SET", run+name, "$", doc).Err(); err != nil {
					t.Fatal(err)
				}
				defer client.Del(ctx, run+name)
			}
			waitIndexed(t, client, index, len(docs))

			c := New(client, index, 48*time.Hour, time.Hour)
			c.cleanupDocuments(ctx)

			for name := range docs {
				exists := client.Exists(ctx, run+name).Val() == 1
				if want := strings.HasSuffix(name, "fresh"); exists != want {
					t.Errorf("%s exists = %v, want %v", name, exists, want)
				}
			}
		})
	}
}

func waitIndexed(t *testing.T, client *redis.Client, index string, n int) {
	t.Helper()
	for i := 0; i < 50; i++ {
		res, err := client.Do(context.Background(), "FT.SEARCH", index, "*", "NOCONTENT", "LIMIT", "0", "0").Slice()
		if err == nil && len(res) > 0 && res[0] == int64(n) {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("index %s never held %d documents", index, n)
}
