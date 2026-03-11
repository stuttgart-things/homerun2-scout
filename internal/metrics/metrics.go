package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// MessagesTotal tracks total messages by severity.
	MessagesTotal = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "scout_messages_total",
		Help: "Total messages indexed in RediSearch by severity",
	}, []string{"severity"})

	// SystemsTotal tracks the number of distinct systems.
	SystemsTotal = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "scout_systems_total",
		Help: "Total distinct systems in RediSearch index",
	})

	// AlertsTotal tracks total alerts by severity.
	AlertsTotal = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "scout_alerts_total",
		Help: "Total alert messages (error/critical) by severity",
	}, []string{"severity"})

	// AggregationDuration tracks how long each aggregation cycle takes.
	AggregationDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "scout_aggregation_duration_seconds",
		Help:    "Duration of aggregation cycles in seconds",
		Buckets: prometheus.DefBuckets,
	})

	// AggregationErrors tracks aggregation errors.
	AggregationErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "scout_aggregation_errors_total",
		Help: "Total number of aggregation errors",
	})
)
