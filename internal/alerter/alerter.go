package alerter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/stuttgart-things/homerun2-scout/internal/models"
)

// ThresholdConfig defines alerting thresholds.
type ThresholdConfig struct {
	ErrorThreshold    int64         // Alert when error count exceeds this
	CriticalThreshold int64         // Alert when critical count exceeds this
	Cooldown          time.Duration // Minimum time between alerts for the same metric
}

// Alerter checks aggregation results against thresholds and sends meta-alerts.
type Alerter struct {
	pitcherURL string
	authToken  string
	thresholds ThresholdConfig
	httpClient *http.Client

	mu          sync.Mutex
	lastAlerted map[string]time.Time
}

// PitchRequest is the payload sent to omni-pitcher.
type PitchRequest struct {
	Title    string `json:"title"`
	Message  string `json:"message"`
	Severity string `json:"severity"`
	Author   string `json:"author"`
	System   string `json:"system"`
	Tags     string `json:"tags"`
}

// New creates a new Alerter.
func New(pitcherURL, authToken string, thresholds ThresholdConfig) *Alerter {
	return &Alerter{
		pitcherURL:  pitcherURL,
		authToken:   authToken,
		thresholds:  thresholds,
		httpClient:  &http.Client{Timeout: 10 * time.Second},
		lastAlerted: make(map[string]time.Time),
	}
}

// Check evaluates the latest aggregation results against thresholds.
func (a *Alerter) Check(ctx context.Context, summary *models.Summary, alerts *models.AlertStats) {
	if a.pitcherURL == "" {
		return
	}

	// Check error threshold
	if errorCount, ok := summary.SeverityCounts["error"]; ok && a.thresholds.ErrorThreshold > 0 {
		if errorCount >= a.thresholds.ErrorThreshold {
			a.sendAlert(ctx, "error-threshold", PitchRequest{
				Title:    "Scout: Error threshold exceeded",
				Message:  fmt.Sprintf("Error count %d exceeds threshold %d", errorCount, a.thresholds.ErrorThreshold),
				Severity: "WARNING",
				Author:   "homerun2-scout",
				System:   "homerun2-scout",
				Tags:     "scout,threshold,error",
			})
		}
	}

	// Check critical threshold
	if criticalCount, ok := summary.SeverityCounts["critical"]; ok && a.thresholds.CriticalThreshold > 0 {
		if criticalCount >= a.thresholds.CriticalThreshold {
			a.sendAlert(ctx, "critical-threshold", PitchRequest{
				Title:    "Scout: Critical threshold exceeded",
				Message:  fmt.Sprintf("Critical count %d exceeds threshold %d", criticalCount, a.thresholds.CriticalThreshold),
				Severity: "CRITICAL",
				Author:   "homerun2-scout",
				System:   "homerun2-scout",
				Tags:     "scout,threshold,critical",
			})
		}
	}

	// Check total alerts spike
	if alerts.TotalAlerts > 0 && a.thresholds.ErrorThreshold > 0 {
		totalThreshold := a.thresholds.ErrorThreshold + a.thresholds.CriticalThreshold
		if totalThreshold > 0 && alerts.TotalAlerts >= totalThreshold {
			a.sendAlert(ctx, "total-alerts-threshold", PitchRequest{
				Title:    "Scout: Total alerts threshold exceeded",
				Message:  fmt.Sprintf("Total alerts %d exceeds combined threshold %d", alerts.TotalAlerts, totalThreshold),
				Severity: "ERROR",
				Author:   "homerun2-scout",
				System:   "homerun2-scout",
				Tags:     "scout,threshold,alerts",
			})
		}
	}
}

func (a *Alerter) sendAlert(ctx context.Context, key string, req PitchRequest) {
	a.mu.Lock()
	if last, ok := a.lastAlerted[key]; ok && time.Since(last) < a.thresholds.Cooldown {
		a.mu.Unlock()
		slog.Debug("alert suppressed (cooldown)", "key", key)
		return
	}
	a.lastAlerted[key] = time.Now()
	a.mu.Unlock()

	body, err := json.Marshal(req)
	if err != nil {
		slog.Error("failed to marshal alert", "error", err)
		return
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, a.pitcherURL+"/pitch", bytes.NewReader(body))
	if err != nil {
		slog.Error("failed to create alert request", "error", err)
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if a.authToken != "" {
		httpReq.Header.Set("Authorization", "Bearer "+a.authToken)
	}

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		slog.Error("failed to send alert to pitcher", "key", key, "error", err)
		return
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 300 {
		slog.Warn("alert pitch returned non-success", "key", key, "status", resp.StatusCode)
		return
	}

	slog.Info("meta-alert sent to pitcher", "key", key, "title", req.Title)
}
