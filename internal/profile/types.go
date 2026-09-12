package profile

// ScoutProfile holds the business logic configuration for homerun2-scout,
// loaded from a ScoutProfile custom resource at startup.
type ScoutProfile struct {
	ScoutInterval string        `json:"scoutInterval,omitempty"`
	Retention     RetentionSpec `json:"retention"`
	Alerting      AlertingSpec  `json:"alerting"`
	Digest        DigestSpec    `json:"digest"`
}

// DigestSpec configures the periodic digest pitched through
// alerting.pitcherURL.
type DigestSpec struct {
	Enabled        bool     `json:"enabled,omitempty"`
	Timezone       string   `json:"timezone,omitempty"`
	Hourly         bool     `json:"hourly,omitempty"`
	DailyAt        string   `json:"dailyAt,omitempty"`
	ExcludeSystems []string `json:"excludeSystems,omitempty"`
	TopSystems     *int     `json:"topSystems,omitempty"`
	System         string   `json:"system,omitempty"`
}

// RetentionSpec configures RediSearch index cleanup.
type RetentionSpec struct {
	Enabled bool   `json:"enabled"`
	TTL     string `json:"ttl,omitempty"`
}

// AlertingSpec configures threshold-based meta-alerting to omni-pitcher.
type AlertingSpec struct {
	PitcherURL        string `json:"pitcherURL,omitempty"`
	PitcherToken      string `json:"pitcherToken,omitempty"`
	ErrorThreshold    int64  `json:"errorThreshold,omitempty"`
	CriticalThreshold int64  `json:"criticalThreshold,omitempty"`
	Cooldown          string `json:"cooldown,omitempty"`
}
