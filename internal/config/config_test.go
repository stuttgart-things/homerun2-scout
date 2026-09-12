package config

import (
	"testing"
	"time"
)

func TestLoadConfigDefaults(t *testing.T) {
	// Clear env to test defaults
	for _, key := range []string{"REDIS_ADDR", "REDIS_PORT", "REDIS_PASSWORD", "REDISEARCH_INDEX", "SCOUT_INTERVAL", "SCOUT_RETENTION_TTL", "SCOUT_RETENTION_ENABLED", "AUTH_TOKEN", "PORT", "LOG_FORMAT", "LOG_LEVEL"} {
		t.Setenv(key, "")
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error: %v", err)
	}

	if cfg.RedisAddr != "localhost" {
		t.Errorf("RedisAddr = %q, want %q", cfg.RedisAddr, "localhost")
	}
	if cfg.RedisPort != "6379" {
		t.Errorf("RedisPort = %q, want %q", cfg.RedisPort, "6379")
	}
	if cfg.RedisearchIndex != "messages" {
		t.Errorf("RedisearchIndex = %q, want %q", cfg.RedisearchIndex, "messages")
	}
	if cfg.ScoutInterval != 60*time.Second {
		t.Errorf("ScoutInterval = %v, want %v", cfg.ScoutInterval, 60*time.Second)
	}
	if cfg.Port != "8080" {
		t.Errorf("Port = %q, want %q", cfg.Port, "8080")
	}
	if cfg.LogFormat != "json" {
		t.Errorf("LogFormat = %q, want %q", cfg.LogFormat, "json")
	}
	if cfg.RetentionTTL != 48*time.Hour {
		t.Errorf("RetentionTTL = %v, want %v", cfg.RetentionTTL, 48*time.Hour)
	}
	if !cfg.RetentionEnabled {
		t.Error("RetentionEnabled should default to true")
	}
}

func TestLoadConfigDigest(t *testing.T) {
	for _, key := range []string{"DIGEST_ENABLED", "DIGEST_TIMEZONE", "DIGEST_HOURLY", "DIGEST_DAILY_AT", "DIGEST_EXCLUDE_SYSTEMS", "DIGEST_TOP_SYSTEMS", "DIGEST_SYSTEM"} {
		t.Setenv(key, "")
	}
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DigestEnabled || cfg.DigestHourly || cfg.DigestDailyAt != "" || cfg.DigestTimezone != "UTC" ||
		cfg.DigestTopSystems != 3 || cfg.DigestSystem != "scout-digest" || cfg.DigestExcludeSystems != nil {
		t.Errorf("digest defaults = %+v", cfg)
	}

	t.Setenv("DIGEST_ENABLED", "true")
	t.Setenv("DIGEST_HOURLY", "true")
	t.Setenv("DIGEST_DAILY_AT", "07:00")
	t.Setenv("DIGEST_TIMEZONE", "Europe/Berlin")
	t.Setenv("DIGEST_EXCLUDE_SYSTEMS", " kubernetes, ,smoke-test ")
	t.Setenv("DIGEST_TOP_SYSTEMS", "5")
	t.Setenv("DIGEST_SYSTEM", "digest")
	cfg, err = LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.DigestEnabled || !cfg.DigestHourly || cfg.DigestDailyAt != "07:00" || cfg.DigestTimezone != "Europe/Berlin" ||
		cfg.DigestTopSystems != 5 || cfg.DigestSystem != "digest" ||
		len(cfg.DigestExcludeSystems) != 2 || cfg.DigestExcludeSystems[0] != "kubernetes" || cfg.DigestExcludeSystems[1] != "smoke-test" {
		t.Errorf("digest from env = %+v", cfg)
	}

	t.Setenv("DIGEST_TOP_SYSTEMS", "-1")
	if _, err := LoadConfig(); err == nil {
		t.Error("a negative DIGEST_TOP_SYSTEMS must be rejected")
	}
}

func TestLoadConfigFromEnv(t *testing.T) {
	t.Setenv("REDIS_ADDR", "redis-host")
	t.Setenv("REDIS_PORT", "6380")
	t.Setenv("REDISEARCH_INDEX", "my-index")
	t.Setenv("SCOUT_INTERVAL", "30s")
	t.Setenv("AUTH_TOKEN", "my-token")
	t.Setenv("PORT", "9090")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error: %v", err)
	}

	if cfg.RedisAddr != "redis-host" {
		t.Errorf("RedisAddr = %q, want %q", cfg.RedisAddr, "redis-host")
	}
	if cfg.RedisPort != "6380" {
		t.Errorf("RedisPort = %q, want %q", cfg.RedisPort, "6380")
	}
	if cfg.RedisearchIndex != "my-index" {
		t.Errorf("RedisearchIndex = %q, want %q", cfg.RedisearchIndex, "my-index")
	}
	if cfg.ScoutInterval != 30*time.Second {
		t.Errorf("ScoutInterval = %v, want %v", cfg.ScoutInterval, 30*time.Second)
	}
	if cfg.AuthToken != "my-token" {
		t.Errorf("AuthToken = %q, want %q", cfg.AuthToken, "my-token")
	}
}

func TestLoadConfigInvalidInterval(t *testing.T) {
	t.Setenv("SCOUT_INTERVAL", "not-a-duration")

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error for invalid SCOUT_INTERVAL, got nil")
	}
}

func TestRedisAddress(t *testing.T) {
	cfg := &Config{RedisAddr: "myhost", RedisPort: "6379"}
	if got := cfg.RedisAddress(); got != "myhost:6379" {
		t.Errorf("RedisAddress() = %q, want %q", got, "myhost:6379")
	}
}
