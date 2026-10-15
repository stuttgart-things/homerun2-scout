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

## Digest

Builds the digest of the window ending now - the hour (`?schedule=hourly`), the day (`?schedule=daily`, default) or seven days (`?schedule=weekly`) - exactly as the periodic digest would pitch it, without pitching it. It works whether or not `digest.enabled` is set, and needs the index to declare `timestamp_unix` NUMERIC (homerun-library v4.5.0+); otherwise it answers 503 with the fix.

`windowIncomplete: true` means retention (`SCOUT_RETENTION_TTL`) has already deleted the oldest part of the window, so the counts are too low; `weekly` needs at least 8 days. The previous window is only compared while retention still holds it (15 days for `weekly`).

```bash
curl "http://localhost:8080/analytics/digest?schedule=hourly" \
  -H "Authorization: Bearer $AUTH_TOKEN"
```

Response (example):

```json
{
  "schedule": "hourly",
  "message": {
    "title": "1h: 42 msgs 3 err 0 crit",
    "message": "hourly digest 2026-09-12 14:37 - 2026-09-12 15:37 (Europe/Berlin)\nMessages: 42 (previous 30, +12)\nErrors: 3 (previous 1, +2), critical: 0 (previous 0, +0)\nSeverities: error 3, success 20, info 19\nTop systems: github 5 (3 alerts), kubernetes 37 (0 alerts)",
    "severity": "warning",
    "author": "homerun2-scout",
    "system": "scout-digest",
    "tags": "digest,hourly",
    "timestamp": "2026-09-12T13:37:00Z"
  },
  "current": {"start": "...", "end": "...", "total": 42, "severities": {"error": 3, "success": 20, "info": 19}, "topSystems": [...]},
  "previous": {"...": "..."}
}
```

## Messages

The messages of the window ending now, for consumers that need more than counts - homerun2-ai-pitcher's summaries of PRs, releases and outages (#90). The window is the same as `/analytics/digest`'s, and the digest's exclusions apply: the digest's own system and `DIGEST_EXCLUDE_SYSTEMS` never show up.

| Parameter | Meaning |
|---|---|
| `schedule` | `hourly`, `daily` (default) or `weekly` |
| `systems` | comma-separated system names, exact match |
| `severity` | comma-separated severities, any case |
| `tags` | comma-separated tags; a message matches with any of them, whole tags only (`release`, `pull_request`, `owner/repo`) |
| `limit` | 1 to 500, default 100: the newest matching messages are kept, `omitted` counts the rest |
| `fields=message` | include the message text, cut at 1000 characters; left out by default |

```bash
curl "http://localhost:8080/analytics/messages?schedule=weekly&systems=homerun2-git-pitcher&tags=release" \
  -H "Authorization: Bearer $AUTH_TOKEN"
```

Response (example), oldest first:

```json
{
  "schedule": "weekly",
  "start": "2026-10-08T07:45:00Z",
  "end": "2026-10-15T07:45:00Z",
  "total": 1,
  "omitted": 0,
  "messages": [
    {
      "timestamp": "2026-10-14T09:12:03Z",
      "system": "homerun2-git-pitcher",
      "severity": "success",
      "title": "Release v0.2.0 on stuttgart-things/homerun2-ai-pitcher",
      "author": "github-actions[bot]",
      "tags": ["github", "release", "stuttgart-things/homerun2-ai-pitcher"],
      "url": "https://github.com/stuttgart-things/homerun2-ai-pitcher/releases/tag/v0.2.0"
    }
  ]
}
```

At most 10000 messages of a window are read, newest first; a window with more answers `truncated: true` and `total` is too low. `windowIncomplete` works as for the digest.

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
