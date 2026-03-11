# Grafana Integration

Scout exposes analytics data as Prometheus metrics at `/metrics`. This is the recommended way to build Grafana dashboards — no additional plugins required.

## Add Scout as a Prometheus Datasource

If Prometheus is already scraping Scout (see [Scrape config](#prometheus-scrape-config)), add it as a datasource in Grafana as usual. If you want to query Scout directly without Prometheus, use the [Grafana Infinity plugin](https://grafana.com/grafana/plugins/yesoreyeram-infinity-datasource/) with the REST API endpoints described in [API Usage](api-usage.md).

## Prometheus Scrape Config

Add Scout to your Prometheus `scrape_configs`:

```yaml
scrape_configs:
  - job_name: homerun2-scout
    static_configs:
      - targets: ["homerun2-scout.homerun2.svc.cluster.local:80"]
    # No auth required for /metrics
```

Or via a `ServiceMonitor` if using the Prometheus Operator:

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: homerun2-scout
  namespace: homerun2
spec:
  selector:
    matchLabels:
      app: homerun2-scout
  endpoints:
    - port: http
      path: /metrics
```

## Available Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `homerun2_scout_messages_total` | Gauge | — | Total messages in RediSearch |
| `homerun2_scout_severity_count` | Gauge | `severity` | Message count per severity |
| `homerun2_scout_system_message_count` | Gauge | `system` | Message count per system (top 20) |
| `homerun2_scout_systems_total` | Gauge | — | Distinct system count |
| `homerun2_scout_alert_count` | Gauge | `severity` | Alert count per severity (error/critical) |
| `homerun2_scout_top_alerting_system_count` | Gauge | `system` | Alert count for top 10 alerting systems |
| `homerun2_scout_aggregation_duration_seconds` | Histogram | — | Aggregation cycle duration |
| `homerun2_scout_aggregation_errors_total` | Counter | — | Aggregation error count |

## PromQL Examples

**Total messages:**
```promql
homerun2_scout_messages_total
```

**Severity breakdown (all severities):**
```promql
homerun2_scout_severity_count
```

**Critical message count:**
```promql
homerun2_scout_severity_count{severity="critical"}
```

**Error + critical combined:**
```promql
sum(homerun2_scout_alert_count)
```

**Top 5 systems by message count:**
```promql
topk(5, homerun2_scout_system_message_count)
```

**Top alerting systems:**
```promql
homerun2_scout_top_alerting_system_count
```

**Aggregation cycle duration (p95):**
```promql
histogram_quantile(0.95, rate(homerun2_scout_aggregation_duration_seconds_bucket[5m]))
```

## Sample Dashboard Panels

### Severity Breakdown (Pie Chart)

- **Type:** Pie chart
- **Query:** `homerun2_scout_severity_count`
- **Legend:** `{{severity}}`

### Total Messages (Stat)

- **Type:** Stat
- **Query:** `homerun2_scout_messages_total`

### Top Systems (Bar Chart)

- **Type:** Bar chart
- **Query:** `homerun2_scout_system_message_count`
- **Legend:** `{{system}}`

### Alert Count Over Time (Time Series)

Scout gauges reflect the state at each aggregation cycle. To see change over time:

```promql
homerun2_scout_alert_count{severity="critical"}
homerun2_scout_alert_count{severity="error"}
```

### Systems with Most Alerts (Bar Chart)

- **Type:** Bar chart
- **Query:** `homerun2_scout_top_alerting_system_count`
- **Legend:** `{{system}}`

## Grafana Alerting

Create a Grafana alert rule to fire when critical message count exceeds a threshold:

```promql
homerun2_scout_severity_count{severity="critical"} > 10
```

Or alert when error count spikes:

```promql
homerun2_scout_alert_count{severity="error"} > 50
```

> Note: Scout also has built-in threshold alerting via omni-pitcher (see [ScoutProfile](scout-profile.md)). Use Grafana alerts for dashboard-level visibility and omni-pitcher alerts for automated incident response.
