package aggregator

import (
	"context"
	"log/slog"
	"strings"
)

// EnsureIndex checks whether the RediSearch index exists and creates it if missing.
// The index is created on JSON documents with the schema that scout queries require.
func (a *Aggregator) EnsureIndex(ctx context.Context) error {
	// Check if index already exists
	_, err := a.client.Do(ctx, "FT.INFO", a.index).Result()
	if err == nil {
		slog.Info("redisearch index already exists", "index", a.index)
		return nil
	}

	// If the error is not "no such index", something else is wrong
	if !strings.Contains(err.Error(), "no such index") {
		return err
	}

	slog.Info("redisearch index not found, creating", "index", a.index)

	// FT.CREATE <index> ON JSON SCHEMA
	// severity and system use TEXT (not TAG) to support FT.AGGREGATE GROUPBY.
	// TAG fields on JSON indexes return only the total count without grouped rows.
	args := []interface{}{
		"FT.CREATE", a.index,
		"ON", "JSON",
		"SCHEMA",
		"$.severity", "AS", "severity", "TEXT",
		"$.system", "AS", "system", "TEXT",
		"$.timestamp", "AS", "timestamp", "TEXT",
		"$.title", "AS", "title", "TEXT",
		"$.message", "AS", "message", "TEXT",
		"$.author", "AS", "author", "TEXT",
		"$.tags", "AS", "tags", "TEXT",
	}

	if err := a.client.Do(ctx, args...).Err(); err != nil {
		return err
	}

	slog.Info("redisearch index created", "index", a.index)
	return nil
}
