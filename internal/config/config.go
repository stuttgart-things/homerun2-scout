package config

import (
	"fmt"
	"log/slog"
	"os"
	"time"
)

// Config holds the application configuration loaded from environment variables.
type Config struct {
	RedisAddr        string
	RedisPort        string
	RedisPassword    string
	RedisearchIndex  string
	ScoutInterval    time.Duration
	RetentionTTL     time.Duration
	RetentionEnabled bool
	AuthToken        string
	Port             string
	LogFormat        string
	LogLevel         string
}

// LoadConfig reads configuration from environment variables with sensible defaults.
func LoadConfig() (*Config, error) {
	cfg := &Config{
		RedisAddr:       getEnv("REDIS_ADDR", "localhost"),
		RedisPort:       getEnv("REDIS_PORT", "6379"),
		RedisPassword:   getEnv("REDIS_PASSWORD", ""),
		RedisearchIndex: getEnv("REDISEARCH_INDEX", "messages"),
		AuthToken:       getEnv("AUTH_TOKEN", ""),
		Port:            getEnv("PORT", "8080"),
		LogFormat:       getEnv("LOG_FORMAT", "json"),
		LogLevel:        getEnv("LOG_LEVEL", "info"),
	}

	intervalStr := getEnv("SCOUT_INTERVAL", "60s")
	d, err := time.ParseDuration(intervalStr)
	if err != nil {
		return nil, fmt.Errorf("invalid SCOUT_INTERVAL %q: %w", intervalStr, err)
	}
	cfg.ScoutInterval = d

	retentionStr := getEnv("SCOUT_RETENTION_TTL", "")
	if retentionStr != "" {
		ttl, err := time.ParseDuration(retentionStr)
		if err != nil {
			return nil, fmt.Errorf("invalid SCOUT_RETENTION_TTL %q: %w", retentionStr, err)
		}
		cfg.RetentionTTL = ttl
		cfg.RetentionEnabled = true
	}

	return cfg, nil
}

// RedisAddress returns the full Redis address (host:port).
func (c *Config) RedisAddress() string {
	return fmt.Sprintf("%s:%s", c.RedisAddr, c.RedisPort)
}

// SetupLogging configures the default slog logger based on config.
func SetupLogging(format, level string) {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: lvl}

	var handler slog.Handler
	if format == "text" {
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	slog.SetDefault(slog.New(handler))
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
