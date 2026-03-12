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
	if p.Alerting.Cooldown != "" {
		d, err := time.ParseDuration(p.Alerting.Cooldown)
		if err != nil {
			return fmt.Errorf("invalid alerting.cooldown %q: %w", p.Alerting.Cooldown, err)
		}
		cfg.AlertCooldown = d
	}

	return nil
}
