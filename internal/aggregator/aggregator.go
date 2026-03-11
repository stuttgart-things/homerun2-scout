package aggregator

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stuttgart-things/homerun2-scout/internal/metrics"
	"github.com/stuttgart-things/homerun2-scout/internal/models"
)

// Aggregator periodically runs FT.AGGREGATE queries and caches results.
type Aggregator struct {
	client   *redis.Client
	index    string
	interval time.Duration

	mu      sync.RWMutex
	summary *models.Summary
	systems *models.SystemStats
	alerts  *models.AlertStats

	cancel context.CancelFunc
	done   chan struct{}
}

// New creates a new Aggregator.
func New(client *redis.Client, index string, interval time.Duration) *Aggregator {
	return &Aggregator{
		client:   client,
		index:    index,
		interval: interval,
		done:     make(chan struct{}),
	}
}

// Start begins the periodic aggregation loop.
func (a *Aggregator) Start(ctx context.Context) {
	ctx, a.cancel = context.WithCancel(ctx)

	// Run immediately on start
	a.runOnce(ctx)

	go func() {
		defer close(a.done)
		ticker := time.NewTicker(a.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				a.runOnce(ctx)
			}
		}
	}()

	slog.Info("aggregator started", "index", a.index, "interval", a.interval)
}

// Stop stops the aggregation loop and waits for it to finish.
func (a *Aggregator) Stop() {
	if a.cancel != nil {
		a.cancel()
		<-a.done
	}
	slog.Info("aggregator stopped")
}

// Summary returns the latest cached summary.
func (a *Aggregator) Summary() *models.Summary {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.summary == nil {
		return &models.Summary{
			SeverityCounts: map[string]int64{},
			LastUpdated:    time.Now(),
		}
	}
	return a.summary
}

// Systems returns the latest cached system stats.
func (a *Aggregator) Systems() *models.SystemStats {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.systems == nil {
		return &models.SystemStats{
			Systems:     []models.SystemCount{},
			LastUpdated: time.Now(),
		}
	}
	return a.systems
}

// Alerts returns the latest cached alert stats.
func (a *Aggregator) Alerts() *models.AlertStats {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.alerts == nil {
		return &models.AlertStats{
			SeverityCounts: map[string]int64{},
			TopSystems:     []models.SystemCount{},
			LastUpdated:    time.Now(),
		}
	}
	return a.alerts
}

func (a *Aggregator) runOnce(ctx context.Context) {
	slog.Debug("running aggregation cycle")
	start := time.Now()

	summary := a.aggregateSummary(ctx)
	systems := a.aggregateSystems(ctx)
	alerts := a.aggregateAlerts(ctx)

	a.mu.Lock()
	a.summary = summary
	a.systems = systems
	a.alerts = alerts
	a.mu.Unlock()

	// Record Prometheus metrics
	duration := time.Since(start).Seconds()
	metrics.AggregationDuration.Observe(duration)

	for sev, count := range summary.SeverityCounts {
		metrics.MessagesTotal.WithLabelValues(sev).Set(float64(count))
	}
	metrics.SystemsTotal.Set(float64(systems.Total))
	for sev, count := range alerts.SeverityCounts {
		metrics.AlertsTotal.WithLabelValues(sev).Set(float64(count))
	}

	slog.Info("aggregation cycle complete",
		"totalMessages", summary.TotalMessages,
		"systemCount", systems.Total,
		"totalAlerts", alerts.TotalAlerts,
		"duration", duration,
	)
}
