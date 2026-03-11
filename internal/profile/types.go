package profile

// ScoutProfile holds the business logic configuration for homerun2-scout,
// loaded from a ScoutProfile custom resource at startup.
type ScoutProfile struct {
	ScoutInterval string        `json:"scoutInterval,omitempty"`
	Retention     RetentionSpec `json:"retention"`
	Alerting      AlertingSpec  `json:"alerting"`
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
