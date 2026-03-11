# API Usage

## Health Check

```bash
curl http://localhost:8080/health
```

Response:

```json
{
  "status": "ok",
  "version": "1.0.0",
  "commit": "abc123",
  "date": "2026-03-11",
  "uptime": "2h30m15s"
}
```

## Analytics Summary

Returns overall message counts grouped by severity.

```bash
curl http://localhost:8080/analytics/summary \
  -H "Authorization: Bearer $AUTH_TOKEN"
```

Response:

```json
{
  "totalMessages": 1542,
  "severityCounts": {
    "info": 1200,
    "warning": 250,
    "error": 80,
    "critical": 12
  },
  "timeWindow": "60s",
  "lastUpdated": "2026-03-11T12:00:00Z"
}
```

## Systems

Returns per-system message counts, sorted by count descending (top 20).

```bash
curl http://localhost:8080/analytics/systems \
  -H "Authorization: Bearer $AUTH_TOKEN"
```

Response:

```json
{
  "systems": [
    {"system": "api-gateway", "count": 500},
    {"system": "worker-pool", "count": 320},
    {"system": "scheduler", "count": 180}
  ],
  "total": 3,
  "lastUpdated": "2026-03-11T12:00:00Z"
}
```

## Alerts

Returns alert statistics (error + critical severity only) with top alerting systems.

```bash
curl http://localhost:8080/analytics/alerts \
  -H "Authorization: Bearer $AUTH_TOKEN"
```

Response:

```json
{
  "totalAlerts": 92,
  "severityCounts": {
    "error": 80,
    "critical": 12
  },
  "topSystems": [
    {"system": "api-gateway", "count": 45},
    {"system": "worker-pool", "count": 30}
  ],
  "lastUpdated": "2026-03-11T12:00:00Z"
}
```

## Error Responses

### 401 Unauthorized

Missing or invalid Bearer token:

```json
{"error": "missing authorization header"}
```

```json
{"error": "invalid token"}
```

### 405 Method Not Allowed

Wrong HTTP method (e.g., POST to a GET endpoint):

```json
{"error": "method not allowed"}
```
