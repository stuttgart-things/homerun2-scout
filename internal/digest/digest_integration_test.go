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

// TestRedisQuerierMessages reads a window's messages page by page against a
// redis-stack:
//
//	REDIS_ADDR=localhost REDIS_PORT=6379 go test -tags integration ./internal/digest/
func TestRedisQuerierMessages(t *testing.T) {
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

	run := fmt.Sprintf("dm%d:", time.Now().UnixNano())
	index := strings.TrimSuffix(run, ":")
	if err := client.Do(ctx, "FT.CREATE", index, "ON", "JSON", "PREFIX", "1", run, "SCHEMA",
		"$.severity", "AS", "severity", "TEXT", "$.system", "AS", "system", "TEXT",
		"$.title", "AS", "title", "TEXT", "$.tags", "AS", "tags", "TEXT",
		"$.timestamp_unix", "AS", "timestamp_unix", "NUMERIC", "SORTABLE").Err(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = client.Do(ctx, "FT.DROPINDEX", index, "DD").Err() }()

	// More than two pages in the window, one message on each side of it.
	end := time.Date(2026, 9, 12, 7, 0, 0, 0, time.UTC)
	start := end.Add(-24 * time.Hour)
	const inWindow = 2*searchBatch + 3
	put := func(key string, at time.Time, title string) {
		doc := fmt.Sprintf(`{"system":"homerun2-git-pitcher","severity":"success","title":%q,"tags":"github,release,o/r","url":"https://x","timestamp":%q,"timestamp_unix":%d}`,
			title, at.Format(time.RFC3339), at.Unix())
		if err := client.Do(ctx, "JSON.SET", run+key, "$", doc).Err(); err != nil {
			t.Fatal(err)
		}
	}
	for i := range inWindow {
		put(fmt.Sprintf("%d", i), start.Add(time.Duration(i)*time.Minute), fmt.Sprintf("msg %d", i))
	}
	put("before", start.Add(-time.Second), "before")
	put("end", end, "at end")
	time.Sleep(time.Second)

	q := RedisQuerier{Client: client, Index: index}
	entries, truncated, err := q.Messages(ctx, start, end, maxScan)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != inWindow || truncated {
		t.Fatalf("%d entries (truncated %v), want %d", len(entries), truncated, inWindow)
	}
	first, last := entries[0], entries[len(entries)-1]
	if first.Title != fmt.Sprintf("msg %d", inWindow-1) || last.Title != "msg 0" {
		t.Errorf("not newest first: %q ... %q", first.Title, last.Title)
	}
	if first.System != "homerun2-git-pitcher" || first.URL != "https://x" || strings.Join(first.Tags, ",") != "github,release,o/r" ||
		!first.Timestamp.Equal(start.Add(time.Duration(inWindow-1)*time.Minute)) {
		t.Errorf("entry %+v", first)
	}

	entries, truncated, err = q.Messages(ctx, start, end, searchBatch+10)
	if err != nil || len(entries) != searchBatch+10 || !truncated {
		t.Errorf("capped: %d entries, truncated %v, err %v", len(entries), truncated, err)
	}

	legacy := index + "-legacy"
	if err := client.Do(ctx, "FT.CREATE", legacy, "ON", "JSON", "PREFIX", "1", run, "SCHEMA", "$.severity", "AS", "severity", "TEXT").Err(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = client.Do(ctx, "FT.DROPINDEX", legacy).Err() }()
	if _, _, err := (RedisQuerier{Client: client, Index: legacy}).Messages(ctx, start, end, maxScan); err != ErrNoNumericTimestamp {
		t.Errorf("legacy index err = %v, want ErrNoNumericTimestamp", err)
	}
}
