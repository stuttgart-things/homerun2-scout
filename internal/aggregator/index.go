package aggregator

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
)

// TimestampUnixField is the Unix-seconds event time homerun-library (v4.5.0+)
// writes into every Redis JSON message next to the RFC3339 timestamp
// (homerun.RediSearchTimestampField). The index declares it NUMERIC, which is
// what makes time windows queryable.
const TimestampUnixField = "timestamp_unix"

// EnsureIndex checks whether the RediSearch index exists and creates it if missing.
// The index is created on JSON documents with the schema that scout queries require.
//
// An existing index keeps the schema it was created with. If it predates the
// numeric timestamp, EnsureIndex logs how to recreate it instead of dropping it:
// homerun2-omni-pitcher creates the same index, and dropping it under running
// services is an operator's decision.
func (a *Aggregator) EnsureIndex(ctx context.Context) error {
	// Check if index already exists
	info, err := a.client.Do(ctx, "FT.INFO", a.index).Result()
	if err == nil {
		slog.Info("redisearch index already exists", "index", a.index)
		if !HasNumericAttribute(info, TimestampUnixField) {
			slog.Warn("redisearch index has no NUMERIC "+TimestampUnixField+": retention falls back to comparing every timestamp in Go",
				"index", a.index,
				"fix", fmt.Sprintf("FT.DROPINDEX %s (without DD, the documents stay), then restart scout to recreate it", a.index))
		}
		return nil
	}

	// If the error is not about a missing index, something else is wrong
	if !isMissingIndexError(err) {
		return err
	}

	slog.Info("redisearch index not found, creating", "index", a.index)

	if err := a.client.Do(ctx, indexCreateArgs(a.index)...).Err(); err != nil {
		return err
	}

	slog.Info("redisearch index created", "index", a.index)
	return nil
}

// indexCreateArgs is the FT.CREATE command for index. homerun2-omni-pitcher
// creates the same index (internal/pitcher/index.go); the two definitions must
// stay identical, since whichever service starts first creates it.
//
// severity and system use TEXT (not TAG) to support FT.AGGREGATE GROUPBY:
// TAG fields on JSON indexes return only the total count without grouped rows.
func indexCreateArgs(index string) []any {
	return []any{
		"FT.CREATE", index,
		"ON", "JSON",
		"SCHEMA",
		"$.severity", "AS", "severity", "TEXT",
		"$.system", "AS", "system", "TEXT",
		"$.timestamp", "AS", "timestamp", "TEXT",
		"$.title", "AS", "title", "TEXT",
		"$.message", "AS", "message", "TEXT",
		"$.author", "AS", "author", "TEXT",
		"$.tags", "AS", "tags", "TEXT",
		"$." + TimestampUnixField, "AS", TimestampUnixField, "NUMERIC", "SORTABLE",
	}
}

// HasNumericAttribute reports whether an FT.INFO reply declares attribute as
// NUMERIC. It reads both reply shapes go-redis returns: RESP2 (what scout
// uses) a flat list whose attributes are flat lists, RESP3 a map whose
// attributes are maps.
func HasNumericAttribute(info any, attribute string) bool {
	for _, a := range infoAttributes(info) {
		fields := map[string]any{}
		switch v := a.(type) {
		case map[any]any:
			for k, val := range v {
				if ks, ok := k.(string); ok {
					fields[ks] = val
				}
			}
		case []any:
			for i := 0; i+1 < len(v); i += 2 {
				if ks, ok := v[i].(string); ok {
					fields[ks] = v[i+1]
				}
			}
		}
		if fields["attribute"] == attribute {
			typ, _ := fields["type"].(string)
			return strings.EqualFold(typ, "NUMERIC")
		}
	}
	return false
}

// infoAttributes returns the attributes list of an FT.INFO reply.
func infoAttributes(info any) []any {
	switch v := info.(type) {
	case map[any]any:
		attrs, _ := v["attributes"].([]any)
		return attrs
	case []any:
		for i := 0; i+1 < len(v); i += 2 {
			if v[i] == "attributes" {
				attrs, _ := v[i+1].([]any)
				return attrs
			}
		}
	}
	return nil
}

// isMissingIndexError reports whether err is RediSearch saying the index does
// not exist. The wording differs between Redis Stack / RediSearch versions.
func isMissingIndexError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "no such index") || strings.Contains(msg, "Unknown index name")
}
