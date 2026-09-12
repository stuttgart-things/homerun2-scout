package profile

import (
	"fmt"
	"time"

	"github.com/stuttgart-things/homerun2-scout/internal/config"
)

// Merge applies non-zero fields from p onto cfg.
// Env var defaults in cfg are preserved when the profile field is empty/zero.
func Merge(cfg *config.Config, p *ScoutProfile) error {
	if p == nil {
		return nil
	}

	if p.ScoutInterval != "" {
		d, err := time.ParseDuration(p.ScoutInterval)
		if err != nil {
			return fmt.Errorf("invalid scoutInterval %q: %w", p.ScoutInterval, err)
		}
		cfg.ScoutInterval = d
	}

	// Retention: CR can override both enabled flag and TTL independently
	cfg.RetentionEnabled = p.Retention.Enabled
	if p.Retention.TTL != "" {
		ttl, err := time.ParseDuration(p.Retention.TTL)
		if err != nil {
			return fmt.Errorf("invalid retention.ttl %q: %w", p.Retention.TTL, err)
		}
		cfg.RetentionTTL = ttl
	}

	if p.Alerting.PitcherURL != "" {
		cfg.AlertPitcherURL = p.Alerting.PitcherURL
	}
	if p.Alerting.PitcherToken != "" {
		cfg.AlertPitcherToken = p.Alerting.PitcherToken
	}
	if p.Alerting.ErrorThreshold != 0 {
		cfg.AlertErrorThreshold = p.Alerting.ErrorThreshold
	}
	if p.Alerting.CriticalThreshold != 0 {
		cfg.AlertCriticalThreshold = p.Alerting.CriticalThreshold
	}
	mergeDigest(cfg, p.Digest)

	if p.Alerting.Cooldown != "" {
		d, err := time.ParseDuration(p.Alerting.Cooldown)
		if err != nil {
			return fmt.Errorf("invalid alerting.cooldown %q: %w", p.Alerting.Cooldown, err)
		}
		cfg.AlertCooldown = d
	}

	return nil
}

// mergeDigest applies the set digest fields. enabled and hourly only switch on:
// a profile without a digest block leaves a digest enabled by env alone.
func mergeDigest(cfg *config.Config, d DigestSpec) {
	if d.Enabled {
		cfg.DigestEnabled = true
	}
	if d.Hourly {
		cfg.DigestHourly = true
	}
	if d.Timezone != "" {
		cfg.DigestTimezone = d.Timezone
	}
	if d.DailyAt != "" {
		cfg.DigestDailyAt = d.DailyAt
	}
	if len(d.ExcludeSystems) > 0 {
		cfg.DigestExcludeSystems = d.ExcludeSystems
	}
	if d.TopSystems != nil {
		cfg.DigestTopSystems = *d.TopSystems
	}
	if d.System != "" {
		cfg.DigestSystem = d.System
	}
}
