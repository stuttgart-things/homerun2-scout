//go:build integration

package digest

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// TestRedisQuerierAndStore runs the digest's Redis side against a redis-stack:
//
//	REDIS_ADDR=localhost REDIS_PORT=6379 go test -tags integration ./internal/digest/
func TestRedisQuerierAndStore(t *testing.T) {
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

	run := fmt.Sprintf("dg%d:", time.Now().UnixNano())
	index := strings.TrimSuffix(run, ":")
	if err := client.Do(ctx, "FT.CREATE", index, "ON", "JSON", "PREFIX", "1", run, "SCHEMA",
		"$.severity", "AS", "severity", "TEXT", "$.system", "AS", "system", "TEXT",
		"$.timestamp_unix", "AS", "timestamp_unix", "NUMERIC", "SORTABLE").Err(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = client.Do(ctx, "FT.DROPINDEX", index, "DD").Err() }()

	end := time.Date(2026, 9, 12, 7, 0, 0, 0, time.UTC)
	docs := []struct {
		system, severity string
		at               time.Time
	}{
		{"github", "ERROR", end.Add(-10 * time.Minute)},
		{"github", "error", end.Add(-20 * time.Minute)},
		{"kubernetes", "SUCCESS", end.Add(-30 * time.Minute)},
		{"scout-digest", "success", end.Add(-5 * time.Minute)},
		{"github", "ERROR", end},                        // the window is [start, end)
		{"github", "ERROR", end.Add(-61 * time.Minute)}, // the hour before
	}
	// More than 10 groups, all of which have to come back.
	for i := range 12 {
		docs = append(docs, struct {
			system, severity string
			at               time.Time
		}{fmt.Sprintf("sys-%02d", i), "INFO", end.Add(-time.Minute)})
	}
	for i, d := range docs {
		doc := fmt.Sprintf(`{"system":%q,"severity":%q,"timestamp_unix":%d}`, d.system, d.severity, d.at.Unix())
		if err := client.Do(ctx, "JSON.SET", fmt.Sprintf("%s%d", run, i), "$", doc).Err(); err != nil {
			t.Fatal(err)
		}
	}
	time.Sleep(time.Second)

	q := RedisQuerier{Client: client, Index: index}
	rows, err := q.Window(ctx, end.Add(-time.Hour), end)
	if err != nil {
		t.Fatal(err)
	}
	sum := Summarize(rows, end.Add(-time.Hour), end, []string{"scout-digest"}, 100)
	if sum.Total != 15 || sum.Errors() != 2 || sum.Severities["success"] != 1 || len(sum.TopSystems) != 14 || sum.TopSystems[0].System != "github" {
		t.Errorf("window summary = %+v (rows %d)", sum, len(rows))
	}

	// An index without the numeric attribute is an error, not a quiet window.
	legacy := index + "-legacy"
	if err := client.Do(ctx, "FT.CREATE", legacy, "ON", "JSON", "PREFIX", "1", run, "SCHEMA", "$.severity", "AS", "severity", "TEXT").Err(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = client.Do(ctx, "FT.DROPINDEX", legacy).Err() }()
	if _, err := (RedisQuerier{Client: client, Index: legacy}).Window(ctx, end.Add(-time.Hour), end); err != ErrNoNumericTimestamp {
		t.Errorf("legacy index err = %v, want ErrNoNumericTimestamp", err)
	}

	store := RedisStore{Client: client}
	key := run + "claim"
	defer client.Del(ctx, key)
	first, err1 := store.Claim(ctx, key, time.Minute)
	second, err2 := store.Claim(ctx, key, time.Minute)
	if !first || second || err1 != nil || err2 != nil {
		t.Errorf("claims = %v %v (%v %v), want only the first to succeed", first, second, err1, err2)
	}
	if err := store.Release(ctx, key); err != nil {
		t.Fatal(err)
	}
	if again, _ := store.Claim(ctx, key, time.Minute); !again {
		t.Error("a released claim must be claimable again")
	}
}
