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

func TestMerge_digest(t *testing.T) {
	cfg := baseConfig()
	cfg.DigestTimezone, cfg.DigestTopSystems, cfg.DigestSystem = "UTC", 3, "scout-digest"
	five := 5
	p := &ScoutProfile{Retention: RetentionSpec{Enabled: true}, Digest: DigestSpec{
		Enabled: true, Timezone: "Europe/Berlin", DailyAt: "07:00", ExcludeSystems: []string{"kubernetes"}, TopSystems: &five,
	}}
	if err := Merge(cfg, p); err != nil {
		t.Fatal(err)
	}
	if !cfg.DigestEnabled || cfg.DigestHourly || cfg.DigestTimezone != "Europe/Berlin" || cfg.DigestDailyAt != "07:00" ||
		cfg.DigestTopSystems != 5 || cfg.DigestSystem != "scout-digest" || len(cfg.DigestExcludeSystems) != 1 {
		t.Errorf("merged digest = %+v", cfg)
	}

	// A profile without a digest block leaves an env-enabled digest alone.
	env := baseConfig()
	env.DigestEnabled, env.DigestHourly, env.DigestTopSystems = true, true, 3
	if err := Merge(env, &ScoutProfile{Retention: RetentionSpec{Enabled: true}}); err != nil {
		t.Fatal(err)
	}
	if !env.DigestEnabled || !env.DigestHourly || env.DigestTopSystems != 3 {
		t.Errorf("empty digest block changed the env config: %+v", env)
	}

	// topSystems: 0 is a choice, not an unset field.
	zero := 0
	z := baseConfig()
	z.DigestTopSystems = 3
	_ = Merge(z, &ScoutProfile{Digest: DigestSpec{TopSystems: &zero}})
	if z.DigestTopSystems != 0 {
		t.Errorf("topSystems 0 = %d", z.DigestTopSystems)
	}
}
