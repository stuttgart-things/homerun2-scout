package models

import "time"

// Summary holds the overall analytics summary.
type Summary struct {
	TotalMessages  int64              `json:"totalMessages"`
	SeverityCounts map[string]int64   `json:"severityCounts"`
	TimeWindow     string             `json:"timeWindow"`
	LastUpdated    time.Time          `json:"lastUpdated"`
}

// SystemStats holds per-system analytics.
type SystemStats struct {
	Systems     []SystemCount `json:"systems"`
	Total       int           `json:"total"`
	LastUpdated time.Time     `json:"lastUpdated"`
}

// SystemCount holds message count for a single system.
type SystemCount struct {
	System string `json:"system"`
	Count  int64  `json:"count"`
}

// AlertStats holds alert-related analytics.
type AlertStats struct {
	TotalAlerts    int64         `json:"totalAlerts"`
	SeverityCounts map[string]int64 `json:"severityCounts"`
	TopSystems     []SystemCount `json:"topSystems"`
	LastUpdated    time.Time     `json:"lastUpdated"`
}

// HealthResponse holds the health check response.
type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
	Uptime  string `json:"uptime"`
}
