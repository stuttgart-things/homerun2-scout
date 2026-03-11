# Homerun2 Scout

A periodic analytics service that analyzes messages indexed in RediSearch and exposes aggregated insights via a REST API.

## Overview

Scout is part of the homerun2 platform. It periodically runs `FT.AGGREGATE` queries against the RediSearch index (populated by omni-pitcher's dual-write) and caches the results for fast API access.

## Quick Start

```bash
# Set required environment variables
export REDIS_ADDR=redis-stack.homerun2.svc.cluster.local
export REDIS_PORT=6379
export REDISEARCH_INDEX=messages
export SCOUT_INTERVAL=60s
export AUTH_TOKEN=your-secret-token

# Run locally
go run main.go
```

## API Endpoints

| Endpoint               | Method | Auth     | Description                              |
|------------------------|--------|----------|------------------------------------------|
| `/health`              | GET    | No       | Health check (version, uptime)           |
| `/analytics/summary`   | GET    | Bearer   | Severity counts, total messages          |
| `/analytics/systems`   | GET    | Bearer   | Per-system message counts, top N         |
| `/analytics/alerts`    | GET    | Bearer   | Alert frequency, top alerting systems    |
| `/metrics`             | GET    | No       | Prometheus metrics (Grafana integration) |

## Authentication

All `/analytics/*` endpoints require a Bearer token:

```bash
curl http://localhost:8080/analytics/summary \
  -H "Authorization: Bearer $AUTH_TOKEN"
```

## Architecture

- **Go** stdlib `net/http` — HTTP server with graceful shutdown
- **RediSearch** — `FT.AGGREGATE` queries for analytics
- **time.Ticker** — Periodic aggregation on configurable interval
- **ScoutProfile CRD** — Kubernetes-native business logic configuration
- **ko** — Container image builds (distroless)
- **KCL** — Kubernetes manifest generation
- **Dagger** — CI/CD pipeline functions

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `REDIS_ADDR` | `localhost` | Redis host |
| `REDIS_PORT` | `6379` | Redis port |
| `REDIS_PASSWORD` | (empty) | Redis password |
| `REDISEARCH_INDEX` | `messages` | RediSearch index name |
| `SCOUT_INTERVAL` | `60s` | Aggregation interval |
| `SCOUT_PROFILE_NAME` | (empty) | ScoutProfile CR name to load at startup |
| `SCOUT_RETENTION_TTL` | (empty) | Retention TTL for index cleanup |
| `AUTH_TOKEN` | (empty) | Bearer token for API auth |
| `PORT` | `8080` | HTTP server port |
| `LOG_FORMAT` | `json` | Log format (`json` or `text`) |
| `LOG_LEVEL` | `info` | Log level (`debug`, `info`, `warn`, `error`) |
| `ALERT_PITCHER_URL` | (empty) | omni-pitcher `/pitch` endpoint |
| `ALERT_PITCHER_TOKEN` | (empty) | Bearer token for omni-pitcher |
| `ALERT_ERROR_THRESHOLD` | `0` | Error count threshold to trigger alert |
| `ALERT_CRITICAL_THRESHOLD` | `0` | Critical count threshold to trigger alert |
| `ALERT_COOLDOWN` | `5m` | Minimum time between alerts |

> When `SCOUT_PROFILE_NAME` is set, the corresponding `ScoutProfile` CR overrides these env var values at startup. See [ScoutProfile](scout-profile.md).

## Grafana

Scout exposes analytics as Prometheus metrics at `/metrics`. See [Grafana Integration](grafana.md) for the full metrics reference, PromQL examples, and sample dashboard panels.

## Prerequisites

- **omni-pitcher** must be running with RediSearch dual-write enabled (`REDIS_SEARCH_INDEX` env var set)
- **Redis Stack** with RediSearch module loaded
