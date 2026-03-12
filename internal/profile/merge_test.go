package profile

import (
	"testing"
	"time"

	"github.com/stuttgart-things/homerun2-scout/internal/config"
)

func baseConfig() *config.Config {
	return &config.Config{
		ScoutInterval:          60 * time.Second,
		RetentionEnabled:       true,
		RetentionTTL:           48 * time.Hour,
		AlertPitcherURL:        "",
		AlertPitcherToken:      "",
		AlertErrorThreshold:    0,
		AlertCriticalThreshold: 0,
		AlertCooldown:          5 * time.Minute,
	}
}

func TestMerge_nil(t *testing.T) {
	cfg := baseConfig()
	if err := Merge(cfg, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ScoutInterval != 60*time.Second {
		t.Error("nil profile should not change config")
	}
}

func TestMerge_full(t *testing.T) {
	cfg := baseConfig()
	p := &ScoutProfile{
		ScoutInterval: "30s",
		Retention: RetentionSpec{
			Enabled: true,
			TTL:     "168h",
		},
		Alerting: AlertingSpec{
			PitcherURL:        "http://pitcher",
			PitcherToken:      "secret",
			ErrorThreshold:    50,
			CriticalThreshold: 10,
			Cooldown:          "10m",
		},
	}
	if err := Merge(cfg, p); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ScoutInterval != 30*time.Second {
		t.Errorf("ScoutInterval: got %v, want 30s", cfg.ScoutInterval)
	}
	if cfg.RetentionTTL != 168*time.Hour {
		t.Errorf("RetentionTTL: got %v, want 168h", cfg.RetentionTTL)
	}
	if !cfg.RetentionEnabled {
		t.Error("RetentionEnabled should be true")
	}
	if cfg.AlertPitcherURL != "http://pitcher" {
		t.Errorf("AlertPitcherURL: got %q", cfg.AlertPitcherURL)
	}
	if cfg.AlertErrorThreshold != 50 {
		t.Errorf("AlertErrorThreshold: got %d", cfg.AlertErrorThreshold)
	}
	if cfg.AlertCooldown != 10*time.Minute {
		t.Errorf("AlertCooldown: got %v", cfg.AlertCooldown)
	}
}

func TestMerge_partial(t *testing.T) {
	cfg := baseConfig()
	p := &ScoutProfile{
		ScoutInterval: "10s",
	}
	if err := Merge(cfg, p); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ScoutInterval != 10*time.Second {
		t.Errorf("ScoutInterval: got %v, want 10s", cfg.ScoutInterval)
	}
	// Unchanged fields
	if cfg.AlertCooldown != 5*time.Minute {
		t.Errorf("AlertCooldown should be unchanged, got %v", cfg.AlertCooldown)
	}
}

func TestMerge_invalidDuration(t *testing.T) {
	cfg := baseConfig()
	p := &ScoutProfile{ScoutInterval: "notaduration"}
	if err := Merge(cfg, p); err == nil {
		t.Error("expected error for invalid duration")
	}
}
