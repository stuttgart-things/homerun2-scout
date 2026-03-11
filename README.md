# homerun2-scout

A Go microservice that periodically analyzes messages indexed in RediSearch and exposes aggregated analytics via a REST API. Part of the [homerun2](https://github.com/stuttgart-things/homerun-library) platform.

[![Build & Test](https://github.com/stuttgart-things/homerun2-scout/actions/workflows/build-test.yaml/badge.svg)](https://github.com/stuttgart-things/homerun2-scout/actions/workflows/build-test.yaml)
[![Docs](https://img.shields.io/badge/docs-pages-blue)](https://stuttgart-things.github.io/homerun2-scout/)

## API Endpoints

| Endpoint | Method | Auth | Description |
|----------|--------|------|-------------|
| `/health` | `GET` | None | Health check (returns version, commit, uptime) |
| `/analytics/summary` | `GET` | Bearer token | Severity counts, total messages |
| `/analytics/systems` | `GET` | Bearer token | Per-system message counts (top 20) |
| `/analytics/alerts` | `GET` | Bearer token | Alert frequency, top alerting systems |

<details>
<summary><b>Analytics summary</b></summary>

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

</details>

<details>
<summary><b>Systems breakdown</b></summary>

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

</details>

<details>
<summary><b>Alerts</b></summary>

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

</details>

## Deployment

<details>
<summary><b>Container image (ko / ghcr.io)</b></summary>

The container image is built with [ko](https://ko.build) on top of `cgr.dev/chainguard/static` and published to GitHub Container Registry.

```bash
# Pull the image
docker pull ghcr.io/stuttgart-things/homerun2-scout:<tag>

# Run with Docker
docker run -p 8080:8080 \
  -e REDIS_ADDR=redis -e REDIS_PORT=6379 \
  -e REDISEARCH_INDEX=messages -e SCOUT_INTERVAL=60s \
  -e AUTH_TOKEN=mysecret \
  ghcr.io/stuttgart-things/homerun2-scout:<tag>
```

</details>

<details>
<summary><b>Deploy to Kubernetes with KCL</b></summary>

KCL manifests in `kcl/` are the source of truth for Kubernetes deployment. The modular KCL modules cover: `deploy.k`, `service.k`, `ingress.k`, `secret.k`, `configmap.k`, `serviceaccount.k`, `namespace.k`, `httproute.k`.

**Render manifests locally:**

```bash
# Render with kcl CLI
kcl run kcl/ -Y tests/kcl-deploy-profile.yaml

# Render via Dagger (non-interactive)
task render-manifests-quick
```

</details>

<details>
<summary><b>Deploy Redis Stack (prerequisite)</b></summary>

Scout requires a Redis Stack instance with the RediSearch module loaded, and [omni-pitcher](https://github.com/stuttgart-things/homerun2-omni-pitcher) configured with `REDIS_SEARCH_INDEX` for dual-write.

```bash
helmfile apply -f \
  git::https://github.com/stuttgart-things/helm.git@database/redis-stack.yaml.gotmpl \
  --state-values-set storageClass=openebs-hostpath \
  --state-values-set password="<REPLACE>" \
  --state-values-set namespace=homerun2
```

</details>

## Development

<details>
<summary><b>Run locally</b></summary>

**1. Forward Redis from the Kubernetes cluster:**

```bash
kubectl port-forward -n homerun2 svc/redis-stack 6379:6379
```

**2. Set environment variables and run:**

```bash
export REDIS_ADDR=localhost
export REDIS_PORT=6379
export REDISEARCH_INDEX=messages
export SCOUT_INTERVAL=10s
export AUTH_TOKEN=test
export LOG_FORMAT=text

go run .
```

The service starts on port `8080`. If Redis is not reachable, it logs a warning and retries on each aggregation cycle — it will not crash.

**3. Test the endpoints:**

```bash
# Health (no auth)
curl http://localhost:8080/health

# Analytics (Bearer token required)
curl -H "Authorization: Bearer test" http://localhost:8080/analytics/summary
curl -H "Authorization: Bearer test" http://localhost:8080/analytics/systems
curl -H "Authorization: Bearer test" http://localhost:8080/analytics/alerts

# Prometheus metrics (no auth)
curl http://localhost:8080/metrics
```

</details>

<details>
<summary><b>Project structure</b></summary>

```
main.go                    # Entrypoint, routing, aggregator, graceful shutdown
internal/
  aggregator/              # Periodic FT.AGGREGATE queries, result caching
  banner/                  # Startup banner (lipgloss)
  config/                  # Env-based config loading, slog setup
  handlers/                # HTTP handlers (analytics, health)
  middleware/              # Auth (Bearer token), request logging
  models/                  # Analytics response structs
kcl/                       # Kubernetes manifests (modular KCL)
dagger/                    # CI functions (Lint, Build, Test, Scan)
.ko.yaml                   # ko build configuration
Taskfile.yaml              # Task runner
```

</details>

<details>
<summary><b>Configuration reference</b></summary>

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | HTTP server port | `8080` |
| `REDIS_ADDR` | Redis server address | `localhost` |
| `REDIS_PORT` | Redis server port | `6379` |
| `REDIS_PASSWORD` | Redis password | (empty) |
| `REDISEARCH_INDEX` | RediSearch index name | `messages` |
| `SCOUT_INTERVAL` | Aggregation interval (Go duration) | `60s` |
| `AUTH_TOKEN` | Bearer token for API auth | (empty = no auth) |
| `LOG_FORMAT` | Log format: `json` or `text` | `json` |
| `LOG_LEVEL` | Log level: `debug`, `info`, `warn`, `error` | `info` |

</details>

<details>
<summary><b>CI/CD and release process</b></summary>

Releases are fully automated via GitHub Actions and [semantic-release](https://semantic-release.gitbook.io/).

**Workflow chain on merge to `main`:**

1. **Build, Push & Scan Container Image** — builds with ko, pushes to `ghcr.io`, scans with Trivy
2. **Release** (triggered on successful image build) — semantic-release creates a GitHub release, stages image, pushes kustomize OCI artifact

**Trigger a release manually:**

```bash
task trigger-release
```

**Branch naming convention:**

- `fix/<issue-number>-<short-description>` — bug fixes (patch)
- `feat/<issue-number>-<short-description>` — new features (minor)
- `test/<issue-number>-<short-description>` — test-only changes (no release)

</details>

## Testing

<details>
<summary><b>Unit tests</b></summary>

Unit tests run without Redis:

```bash
go test ./...
```

</details>

<details>
<summary><b>Integration tests (Dagger + Redis)</b></summary>

Integration tests spin up a Redis service via Dagger:

```bash
task build-test-binary
```

</details>

<details>
<summary><b>Lint</b></summary>

```bash
task lint
```

</details>

<details>
<summary><b>Build and scan container image</b></summary>

```bash
task build-scan-image-ko
```

</details>

## Links

- [GitHub Pages](https://stuttgart-things.github.io/homerun2-scout/)
- [Releases](https://github.com/stuttgart-things/homerun2-scout/releases)
- [Container Images](https://github.com/stuttgart-things/homerun2-scout/pkgs/container/homerun2-scout)
- [homerun-library](https://github.com/stuttgart-things/homerun-library)
- [homerun2-omni-pitcher](https://github.com/stuttgart-things/homerun2-omni-pitcher) (prerequisite)

## License

See [LICENSE](LICENSE) file.
