package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// MessagesTotal tracks the total message count across all severities.
	MessagesTotal = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "homerun2_scout_messages_total",
		Help: "Total messages indexed in RediSearch",
	})

	// SeverityCount tracks message counts per severity label.
	SeverityCount = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "homerun2_scout_severity_count",
		Help: "Message count per severity",
	}, []string{"severity"})

	// SystemMessageCount tracks message counts per system.
	SystemMessageCount = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "homerun2_scout_system_message_count",
		Help: "Message count per system",
	}, []string{"system"})

	// SystemsTotal tracks the number of distinct systems.
	SystemsTotal = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "homerun2_scout_systems_total",
		Help: "Total distinct systems in RediSearch index",
	})

	// AlertCount tracks alert message counts (error/critical) per severity.
	AlertCount = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "homerun2_scout_alert_count",
		Help: "Alert message count per severity (error/critical)",
	}, []string{"severity"})

	// TopAlertingSystemCount tracks alert counts for the top alerting systems.
	TopAlertingSystemCount = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "homerun2_scout_top_alerting_system_count",
		Help: "Alert count for top alerting systems",
	}, []string{"system"})

	// AggregationDuration tracks how long each aggregation cycle takes.
	AggregationDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "homerun2_scout_aggregation_duration_seconds",
		Help:    "Duration of aggregation cycles in seconds",
		Buckets: prometheus.DefBuckets,
	})

	// AggregationErrors tracks aggregation errors.
	AggregationErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "homerun2_scout_aggregation_errors_total",
		Help: "Total number of aggregation errors",
	})
)
