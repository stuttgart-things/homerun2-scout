package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"github.com/stuttgart-things/homerun2-scout/internal/aggregator"
	"github.com/stuttgart-things/homerun2-scout/internal/banner"
	"github.com/stuttgart-things/homerun2-scout/internal/config"
	"github.com/stuttgart-things/homerun2-scout/internal/handlers"
	"github.com/stuttgart-things/homerun2-scout/internal/middleware"
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

	// Connect to Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddress(),
		Password: cfg.RedisPassword,
	})

	// Health check Redis
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		slog.Warn("redis not reachable at startup, aggregator will retry", "addr", cfg.RedisAddress(), "error", err)
	} else {
		slog.Info("redis connected", "addr", cfg.RedisAddress())
	}

	// Start aggregator
	agg := aggregator.New(rdb, cfg.RedisearchIndex, cfg.ScoutInterval)
	agg.Start(ctx)

	// Setup routes
	mux := http.NewServeMux()

	// Health endpoint (no auth)
	mux.HandleFunc("/health", middleware.LoggingMiddleware(
		handlers.NewHealthHandler(version, commit, date, startTime),
	))

	// Analytics endpoints (with auth)
	authWrap := func(h http.HandlerFunc) http.HandlerFunc {
		return middleware.LoggingMiddleware(middleware.TokenAuthMiddleware(cfg.AuthToken, h))
	}

	mux.HandleFunc("/analytics/summary", authWrap(handlers.NewSummaryHandler(agg)))
	mux.HandleFunc("/analytics/systems", authWrap(handlers.NewSystemsHandler(agg)))
	mux.HandleFunc("/analytics/alerts", authWrap(handlers.NewAlertsHandler(agg)))

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

	// Stop aggregator
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
