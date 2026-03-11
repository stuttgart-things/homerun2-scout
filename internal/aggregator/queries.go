package aggregator

import (
	"context"
	"log/slog"
	"strconv"
	"time"

	"github.com/stuttgart-things/homerun2-scout/internal/models"
)

func (a *Aggregator) aggregateSummary(ctx context.Context) *models.Summary {
	summary := &models.Summary{
		SeverityCounts: map[string]int64{},
		TimeWindow:     a.interval.String(),
		LastUpdated:    time.Now(),
	}

	// FT.AGGREGATE <index> * GROUPBY 1 @severity REDUCE COUNT 0 AS count
	args := []interface{}{
		"FT.AGGREGATE", a.index, "*",
		"GROUPBY", "1", "@severity",
		"REDUCE", "COUNT", "0", "AS", "count",
	}

	result, err := a.client.Do(ctx, args...).Result()
	if err != nil {
		slog.Warn("failed to aggregate severity counts", "error", err)
		return summary
	}

	rows := parseAggregateResult(result)
	for _, row := range rows {
		severity := row["severity"]
		countStr := row["count"]
		count, _ := strconv.ParseInt(countStr, 10, 64)
		if severity != "" {
			summary.SeverityCounts[severity] = count
			summary.TotalMessages += count
		}
	}

	return summary
}

func (a *Aggregator) aggregateSystems(ctx context.Context) *models.SystemStats {
	stats := &models.SystemStats{
		Systems:     []models.SystemCount{},
		LastUpdated: time.Now(),
	}

	// FT.AGGREGATE <index> * GROUPBY 1 @system REDUCE COUNT 0 AS count SORTBY 2 @count DESC MAX 20
	args := []interface{}{
		"FT.AGGREGATE", a.index, "*",
		"GROUPBY", "1", "@system",
		"REDUCE", "COUNT", "0", "AS", "count",
		"SORTBY", "2", "@count", "DESC",
		"MAX", "20",
	}

	result, err := a.client.Do(ctx, args...).Result()
	if err != nil {
		slog.Warn("failed to aggregate system counts", "error", err)
		return stats
	}

	rows := parseAggregateResult(result)
	for _, row := range rows {
		system := row["system"]
		countStr := row["count"]
		count, _ := strconv.ParseInt(countStr, 10, 64)
		if system != "" {
			stats.Systems = append(stats.Systems, models.SystemCount{
				System: system,
				Count:  count,
			})
		}
	}
	stats.Total = len(stats.Systems)

	return stats
}

func (a *Aggregator) aggregateAlerts(ctx context.Context) *models.AlertStats {
	stats := &models.AlertStats{
		SeverityCounts: map[string]int64{},
		TopSystems:     []models.SystemCount{},
		LastUpdated:    time.Now(),
	}

	// Count alerts by severity (error + critical only)
	for _, sev := range []string{"error", "critical"} {
		args := []interface{}{
			"FT.AGGREGATE", a.index,
			"@severity:{" + sev + "}",
			"GROUPBY", "0",
			"REDUCE", "COUNT", "0", "AS", "count",
		}

		result, err := a.client.Do(ctx, args...).Result()
		if err != nil {
			slog.Warn("failed to aggregate alert counts", "severity", sev, "error", err)
			continue
		}

		rows := parseAggregateResult(result)
		for _, row := range rows {
			countStr, _ := row["count"]
			count, _ := strconv.ParseInt(countStr, 10, 64)
			stats.SeverityCounts[sev] = count
			stats.TotalAlerts += count
		}
	}

	// Top systems with error/critical
	args := []interface{}{
		"FT.AGGREGATE", a.index,
		"@severity:{error|critical}",
		"GROUPBY", "1", "@system",
		"REDUCE", "COUNT", "0", "AS", "count",
		"SORTBY", "2", "@count", "DESC",
		"MAX", "10",
	}

	result, err := a.client.Do(ctx, args...).Result()
	if err != nil {
		slog.Warn("failed to aggregate top alert systems", "error", err)
		return stats
	}

	rows := parseAggregateResult(result)
	for _, row := range rows {
		system, _ := row["system"]
		countStr, _ := row["count"]
		count, _ := strconv.ParseInt(countStr, 10, 64)
		if system != "" {
			stats.TopSystems = append(stats.TopSystems, models.SystemCount{
				System: system,
				Count:  count,
			})
		}
	}

	return stats
}

// parseAggregateResult parses the FT.AGGREGATE response into a slice of key-value maps.
// FT.AGGREGATE returns: [count, [key, val, key, val, ...], [key, val, ...], ...]
func parseAggregateResult(result interface{}) []map[string]string {
	arr, ok := result.([]interface{})
	if !ok || len(arr) < 1 {
		return nil
	}

	var rows []map[string]string
	// First element is the count, remaining are rows
	for i := 1; i < len(arr); i++ {
		row, ok := arr[i].([]interface{})
		if !ok {
			continue
		}
		m := make(map[string]string)
		for j := 0; j+1 < len(row); j += 2 {
			key, _ := row[j].(string)
			val, _ := row[j+1].(string)
			m[key] = val
		}
		rows = append(rows, m)
	}

	return rows
}
