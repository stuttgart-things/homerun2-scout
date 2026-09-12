package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata" // the digest's timezone: the chainguard static image ships no zoneinfo

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"github.com/stuttgart-things/homerun2-scout/internal/aggregator"
	"github.com/stuttgart-things/homerun2-scout/internal/alerter"
	"github.com/stuttgart-things/homerun2-scout/internal/banner"
	"github.com/stuttgart-things/homerun2-scout/internal/config"
	"github.com/stuttgart-things/homerun2-scout/internal/digest"
	"github.com/stuttgart-things/homerun2-scout/internal/handlers"
	"github.com/stuttgart-things/homerun2-scout/internal/middleware"
	"github.com/stuttgart-things/homerun2-scout/internal/profile"
	"github.com/stuttgart-things/homerun2-scout/internal/retention"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	startTime := time.Now()

	// Load config
	cfg, err := config.LoadConfig()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Setup logging
	config.SetupLogging(cfg.LogFormat, cfg.LogLevel)

	// Print banner
	banner.Print(version, commit, date)

	// Load ScoutProfile CR (if configured) and merge into config
	if cfg.ScoutProfileName != "" {
		ctx := context.Background()
		ns := os.Getenv("POD_NAMESPACE")
		if ns == "" {
			ns = "homerun2"
		}
		loader, lerr := profile.NewKubernetesLoader()
		if lerr != nil {
			slog.Warn("failed to build k8s client for ScoutProfile, using env defaults", "error", lerr)
		} else {
			p, lerr := loader.Load(ctx, ns, cfg.ScoutProfileName)
			if lerr != nil {
				slog.Warn("could not load ScoutProfile, using env defaults", "name", cfg.ScoutProfileName, "error", lerr)
			} else if merr := profile.Merge(cfg, p); merr != nil {
				slog.Warn("ScoutProfile merge error, using env defaults", "error", merr)
			} else {
				slog.Info("ScoutProfile applied", "name", cfg.ScoutProfileName, "namespace", ns)
			}
		}
	}

	// Connect to Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddress(),
		Password: cfg.RedisPassword,
		Protocol: 2, // Force RESP2 for RediSearch FT.AGGREGATE compatibility
	})

	// Health check Redis. Informational only, and bounded: the aggregator
	// retries the index and its queries every cycle (#73).
	ctx := context.Background()
	pingCtx, cancelPing := context.WithTimeout(ctx, 5*time.Second)
	if err := rdb.Ping(pingCtx).Err(); err != nil {
		slog.Warn("redis not reachable at startup, aggregator will retry", "addr", cfg.RedisAddress(), "error", err)
	} else {
		slog.Info("redis connected", "addr", cfg.RedisAddress())
	}
	cancelPing()

	// Start aggregator
	agg := aggregator.New(rdb, cfg.RedisearchIndex, cfg.ScoutInterval)

	// Setup threshold alerter (if configured)
	if cfg.AlertPitcherURL != "" {
		a := alerter.New(cfg.AlertPitcherURL, cfg.AlertPitcherToken, alerter.ThresholdConfig{
			ErrorThreshold:    cfg.AlertErrorThreshold,
			CriticalThreshold: cfg.AlertCriticalThreshold,
			Cooldown:          cfg.AlertCooldown,
		})
		agg.SetOnCycleCallback(a.Check)
		slog.Info("threshold alerting enabled", "pitcherURL", cfg.AlertPitcherURL)
	}

	agg.Start(ctx)

	// Start retention cleaner (if enabled)
	var cleaner *retention.Cleaner
	if cfg.RetentionEnabled {
		cleaner = retention.New(rdb, cfg.RedisearchIndex, cfg.RetentionTTL, cfg.ScoutInterval)
		cleaner.Start(ctx)
	}

	// Digest: built for /analytics/digest in any case, pitched only when enabled.
	digestRunner := newDigestRunner(cfg, rdb)
	if cfg.DigestEnabled {
		switch {
		case cfg.AlertPitcherURL == "":
			slog.Warn("digest enabled but no alerting.pitcherURL / ALERT_PITCHER_URL: nothing to pitch to")
		case len(digestSchedules(cfg)) == 0:
			slog.Warn("digest enabled but no schedule: set digest.hourly or digest.dailyAt")
		default:
			digestRunner.Start(ctx)
		}
	}

	// Setup routes
	mux := http.NewServeMux()

	// Root endpoint (no auth) — service info and available routes
	mux.HandleFunc("GET /{$}", middleware.LoggingMiddleware(
		handlers.NewRootHandler(version),
	))

	// Health endpoint (no auth)
	mux.HandleFunc("/health", middleware.LoggingMiddleware(
		handlers.NewHealthHandler(version, commit, date, startTime),
	))

	// Readiness (no auth): index exists and aggregation succeeded recently (#75)
	mux.HandleFunc("/ready", middleware.LoggingMiddleware(handlers.NewReadyHandler(agg)))

	// Analytics endpoints (with auth)
	authWrap := func(h http.HandlerFunc) http.HandlerFunc {
		return middleware.LoggingMiddleware(middleware.TokenAuthMiddleware(cfg.AuthToken, h))
	}

	mux.HandleFunc("/analytics/summary", authWrap(handlers.NewSummaryHandler(agg)))
	mux.HandleFunc("/analytics/systems", authWrap(handlers.NewSystemsHandler(agg)))
	mux.HandleFunc("/analytics/alerts", authWrap(handlers.NewAlertsHandler(agg)))
	mux.HandleFunc("/analytics/digest", authWrap(handlers.NewDigestHandler(digestRunner)))

	// Prometheus metrics endpoint (no auth)
	mux.Handle("/metrics", promhttp.Handler())

	// Start server
	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: mux,
	}

	go func() {
		slog.Info("server starting", "port", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down...")

	// Stop retention cleaner
	if cleaner != nil {
		cleaner.Stop()
	}

	// Stop digest and aggregator
	digestRunner.Stop()
	agg.Stop()

	// Shutdown HTTP server
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("server shutdown error", "error", err)
	}

	// Close Redis
	_ = rdb.Close()

	slog.Info("shutdown complete")
}

// digestSchedules returns the configured digest schedules; an invalid daily
// time is logged and left out.
func digestSchedules(cfg *config.Config) []digest.Schedule {
	var schedules []digest.Schedule
	if cfg.DigestHourly {
		schedules = append(schedules, digest.Hourly())
	}
	if cfg.DigestDailyAt != "" {
		daily, err := digest.Daily(cfg.DigestDailyAt)
		if err != nil {
			slog.Warn("digest: daily schedule left out", "error", err)
		} else {
			schedules = append(schedules, daily)
		}
	}
	return schedules
}

// newDigestRunner builds the digest runner from cfg.
func newDigestRunner(cfg *config.Config, rdb *redis.Client) *digest.Runner {
	loc, err := time.LoadLocation(cfg.DigestTimezone)
	if err != nil {
		slog.Warn("digest: unknown timezone, using UTC", "timezone", cfg.DigestTimezone, "error", err)
		loc = time.UTC
	}
	retention := cfg.RetentionTTL
	if !cfg.RetentionEnabled {
		retention = 0
	}
	return digest.NewRunner(digest.Config{
		Schedules:  digestSchedules(cfg),
		Location:   loc,
		Exclude:    cfg.DigestExcludeSystems,
		TopSystems: cfg.DigestTopSystems,
		System:     cfg.DigestSystem,
		Retention:  retention,
	},
		digest.RedisQuerier{Client: rdb, Index: cfg.RedisearchIndex},
		digest.RedisStore{Client: rdb},
		digest.HTTPPitcher{URL: cfg.AlertPitcherURL, Token: cfg.AlertPitcherToken},
	)
}
