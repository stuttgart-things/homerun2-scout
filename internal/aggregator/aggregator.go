package aggregator

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stuttgart-things/homerun2-scout/internal/metrics"
	"github.com/stuttgart-things/homerun2-scout/internal/models"
)

// CycleCallback is called after each aggregation cycle with the latest results.
type CycleCallback func(ctx context.Context, summary *models.Summary, alerts *models.AlertStats)

// Aggregator periodically runs FT.AGGREGATE queries and caches results.
type Aggregator struct {
	client   *redis.Client
	index    string
	interval time.Duration

	mu      sync.RWMutex
	summary *models.Summary
	systems *models.SystemStats
	alerts  *models.AlertStats

	onCycle CycleCallback
	cancel  context.CancelFunc
	done    chan struct{}

	// indexReady is false until the RediSearch index is known to exist, and
	// again whenever a query reports it missing. ensureIndex is EnsureIndex,
	// swappable in tests.
	indexReady  atomic.Bool
	ensureIndex func(ctx context.Context) error

	// Readiness bookkeeping for /ready (#75). cycleErrors counts the failed
	// queries of the running cycle; lastSuccess is the unix-nano time of the
	// last cycle without any, 0 before the first.
	cycleErrors         atomic.Int64
	lastSuccess         atomic.Int64
	consecutiveFailures atomic.Int64
	lastError           atomic.Value // string
	now                 func() time.Time
}

// New creates a new Aggregator.
func New(client *redis.Client, index string, interval time.Duration) *Aggregator {
	a := &Aggregator{
		client:   client,
		index:    index,
		interval: interval,
		done:     make(chan struct{}),
		now:      time.Now,
	}
	a.ensureIndex = a.EnsureIndex
	return a
}

// SetOnCycleCallback sets a callback invoked after each aggregation cycle.
func (a *Aggregator) SetOnCycleCallback(cb CycleCallback) {
	a.onCycle = cb
}

// Start begins the periodic aggregation loop.
func (a *Aggregator) Start(ctx context.Context) {
	ctx, a.cancel = context.WithCancel(ctx)

	go func() {
		defer close(a.done)

		// The first cycle runs here rather than before Start returns: with
		// Redis still starting, index creation and the queries held up the
		// HTTP server behind them (#73).
		a.runOnce(ctx)

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

	a.cycleErrors.Store(0)
	a.ensureIndexIfMissing(ctx)

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

	metrics.MessagesTotal.Set(float64(summary.TotalMessages))
	for sev, count := range summary.SeverityCounts {
		metrics.SeverityCount.WithLabelValues(sev).Set(float64(count))
	}

	metrics.SystemsTotal.Set(float64(systems.Total))
	metrics.SystemMessageCount.Reset()
	for _, sc := range systems.Systems {
		metrics.SystemMessageCount.WithLabelValues(sc.System).Set(float64(sc.Count))
	}

	for sev, count := range alerts.SeverityCounts {
		metrics.AlertCount.WithLabelValues(sev).Set(float64(count))
	}
	metrics.TopAlertingSystemCount.Reset()
	for _, sc := range alerts.TopSystems {
		metrics.TopAlertingSystemCount.WithLabelValues(sc.System).Set(float64(sc.Count))
	}

	a.recordCycle()

	// Invoke callback if set
	if a.onCycle != nil {
		a.onCycle(ctx, summary, alerts)
	}

	slog.Info("aggregation cycle complete",
		"totalMessages", summary.TotalMessages,
		"systemCount", systems.Total,
		"totalAlerts", alerts.TotalAlerts,
		"duration", duration,
	)
}

// ensureIndexIfMissing creates the RediSearch index unless it is known to
// exist. This used to happen once, in Start: a scout that started while Redis
// was still coming up then failed every later cycle with "no such index" until
// the pod was restarted, 15 hours on labda-dev-a (#73). A failure is logged and
// retried on the next cycle.
func (a *Aggregator) ensureIndexIfMissing(ctx context.Context) {
	if a.indexReady.Load() {
		return
	}
	if err := a.ensureIndex(ctx); err != nil {
		slog.Warn("failed to ensure redisearch index, retrying next cycle", "index", a.index, "error", err)
		a.cycleErrors.Add(1)
		a.lastError.Store(err.Error())
		return
	}
	a.indexReady.Store(true)
}

// noteQueryError marks the index missing when a query says so, so the next
// cycle re-creates it: an index can disappear after scout has started, e.g.
// when Redis restarts without it.
func (a *Aggregator) noteQueryError(err error) {
	if err == nil {
		return
	}
	a.cycleErrors.Add(1)
	a.lastError.Store(err.Error())
	if isMissingIndexError(err) && a.indexReady.Swap(false) {
		slog.Warn("redisearch index is gone, re-creating it next cycle", "index", a.index)
	}
}

// recordCycle closes a cycle for readiness: without failed queries (and with the
// index in place) it counts as a success, otherwise as one more failure.
func (a *Aggregator) recordCycle() {
	if a.cycleErrors.Load() == 0 && a.indexReady.Load() {
		a.lastSuccess.Store(a.now().UnixNano())
		a.consecutiveFailures.Store(0)
		a.lastError.Store("") // a healthy cycle should not read like a broken one
		return
	}
	a.consecutiveFailures.Add(1)
}

// staleAfter is how long the last successful cycle may lie back before scout
// stops reporting ready: three intervals, so one failed cycle (a Redis blip)
// does not take scout out of its Service, while an aggregation that has stopped
// working does within minutes rather than never (#75).
func (a *Aggregator) staleAfter() time.Duration {
	return 3 * a.interval
}

// Readiness reports whether scout can currently aggregate, for /ready.
func (a *Aggregator) Readiness() (bool, models.ReadinessResponse) {
	resp := models.ReadinessResponse{
		Status:              "ready",
		IndexReady:          a.indexReady.Load(),
		ConsecutiveFailures: a.consecutiveFailures.Load(),
		StaleAfter:          a.staleAfter().String(),
	}
	if e, ok := a.lastError.Load().(string); ok {
		resp.LastError = e
	}
	last := a.lastSuccess.Load()
	if last != 0 {
		resp.LastSuccess = time.Unix(0, last).UTC().Format(time.RFC3339)
	}

	switch {
	case !resp.IndexReady:
		resp.Reason = "redisearch index not ready"
	case last == 0:
		resp.Reason = "no successful aggregation cycle yet"
	case a.now().Sub(time.Unix(0, last)) > a.staleAfter():
		resp.Reason = "last successful aggregation cycle is older than staleAfter"
	default:
		return true, resp
	}
	resp.Status = "not ready"
	return false, resp
}
