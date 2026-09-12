# ScoutProfile

`ScoutProfile` is a Kubernetes Custom Resource (`homerun2.stuttgart-things.com/v1alpha1`) that holds the scout's business logic configuration. It separates **deployment config** (image, replicas, ingress — managed by KCL profiles) from **runtime behaviour config** (thresholds, retention, alerting — managed per cluster via the CR).

This pattern is designed to be reused across other homerun2 services.

## Schema

```yaml
apiVersion: homerun2.stuttgart-things.com/v1alpha1
kind: ScoutProfile
metadata:
  name: default
  namespace: homerun2
spec:
  scoutInterval: 60s        # Aggregation interval (Go duration)
  retention:
    enabled: true           # Enable periodic cleanup of expired JSON docs + Redis Stream trimming
    ttl: 48h                # Max age of entries to keep (Go duration, default: 48h)
  alerting:
    pitcherURL: https://...  # omni-pitcher base URL; scout appends /pitch
    pitcherToken: ""         # Bearer token — prefer ALERT_PITCHER_TOKEN env var
    errorThreshold: 50       # Error count that triggers a meta-alert
    criticalThreshold: 10    # Critical count that triggers a meta-alert
    cooldown: 5m             # Minimum time between alerts (Go duration)
  digest:                    # Periodic digests, pitched through alerting.pitcherURL
    enabled: true
    timezone: Europe/Berlin  # IANA timezone the windows end in (default UTC)
    hourly: false            # A digest of every full hour
    dailyAt: "07:00"         # A digest of the last day at this local time
    excludeSystems: []       # Systems left out of the counts
    topSystems: 3            # Systems listed per digest
    system: scout-digest     # System the digest is pitched as
```

## How It Works

1. At startup, if `SCOUT_PROFILE_NAME` is set, the scout reads the named CR from the pod's namespace via the Kubernetes API
2. Non-empty fields in the CR **override** the corresponding env var values in config
3. If the CR is missing, unreachable, or `SCOUT_PROFILE_NAME` is empty, the scout starts normally with env var defaults — no crash
4. While running, the scout reads the CR again every `SCOUT_INTERVAL`. When it was created, changed or deleted since startup, the scout shuts down gracefully - logging `restarting to apply the ScoutProfile` with the reason - and Kubernetes restarts the container, which applies the new profile. A read that fails for another reason (API server unreachable, RBAC) changes nothing, so a CR applied after the pod started - e.g. in a later Argo CD sync-wave - no longer waits for the next manual restart

## Apply the CRD

The `ScoutProfile` CRD is included in the rendered KCL manifests and applied alongside the deployment:

```bash
dagger call -m github.com/stuttgart-things/dagger/kcl@v0.82.0 run \
  --source kcl \
  --parameters-file tests/kcl-movie-scripts-profile.yaml \
  export --path /tmp/rendered-homerun2-scout.yaml

kubectl apply -f /tmp/rendered-homerun2-scout.yaml
```

## Create a ScoutProfile

A sample CR for the movie-scripts cluster is in `tests/scout-profile-movie-scripts.yaml`:

```bash
kubectl apply -f tests/scout-profile-movie-scripts.yaml
```

Custom example:

```yaml
apiVersion: homerun2.stuttgart-things.com/v1alpha1
kind: ScoutProfile
metadata:
  name: default
  namespace: homerun2
spec:
  scoutInterval: 30s
  retention:
    enabled: true
    ttl: 72h
  alerting:
    pitcherURL: https://homerun2-omni-pitcher.movie-scripts2.sthings-vsphere.labul.sva.de
    errorThreshold: 20
    criticalThreshold: 5
    cooldown: 10m
```

## Activation

Set `SCOUT_PROFILE_NAME` in the deployment (the KCL `scoutProfileName` parameter, default: `default`):

```yaml
# In KCL profile
config.scoutProfileName: default
```

Or directly via env var when running locally with cluster access:

```bash
export SCOUT_PROFILE_NAME=default
export KUBECONFIG=~/.kube/movie-scripts
go run .
```

## RBAC

The KCL manifests include a `Role` and `RoleBinding` that grant the scout's `ServiceAccount` permission to `get` ScoutProfile CRs in its namespace. No additional RBAC setup is required.

## Local Development (no Kubernetes)

Leave `SCOUT_PROFILE_NAME` unset — profile loading is skipped entirely and all configuration comes from env vars:

```bash
# No SCOUT_PROFILE_NAME → env vars only
export ALERT_PITCHER_URL=http://localhost:8081
export ALERT_ERROR_THRESHOLD=50
export ALERT_CRITICAL_THRESHOLD=10
export SCOUT_RETENTION_TTL=48h
go run .
```

## Field Reference

### `spec.scoutInterval`

Go duration string (e.g. `30s`, `2m`, `1h`). Overrides `SCOUT_INTERVAL` env var.

### `spec.retention.enabled`

Boolean. Enables periodic cleanup of expired JSON documents (via `FT.SEARCH` + `DEL`) and Redis Stream trimming (via `XTRIM MINID`). Default: `true`.

### `spec.retention.ttl`

Go duration string (e.g. `48h` = 2 days). Overrides `SCOUT_RETENTION_TTL` env var. Default: `48h`.

### `spec.alerting.pitcherURL`

Base URL of omni-pitcher, without `/pitch`: scout posts meta-alerts to `<pitcherURL>/pitch`. Overrides `ALERT_PITCHER_URL` env var.

### `spec.alerting.pitcherToken`

Bearer token for omni-pitcher. It is recommended to leave this empty in the CR and use the `ALERT_PITCHER_TOKEN` env var (sourced from a Kubernetes Secret) instead.

### `spec.alerting.errorThreshold`

Integer. Number of `error`-severity messages in a single aggregation cycle that triggers a meta-alert. Overrides `ALERT_ERROR_THRESHOLD` env var.

### `spec.alerting.criticalThreshold`

Integer. Number of `critical`-severity messages that triggers a meta-alert. Overrides `ALERT_CRITICAL_THRESHOLD` env var.

### `spec.alerting.cooldown`

Go duration string. Minimum time between successive alerts to avoid alert storms. Overrides `ALERT_COOLDOWN` env var.

## Digest

With `spec.digest.enabled`, scout pitches a summary of every finished window to omni-pitcher (`<alerting.pitcherURL>/pitch`, token from `ALERT_PITCHER_TOKEN`), so it reaches the same catchers as the messages it counts.

- **Windows**: `hourly` ends on every full local hour; `dailyAt` ends every day at that local time and covers the calendar day before - 23 or 25 hours across a daylight saving change. Times are in `timezone`.
- **Counts**: messages per severity (case-insensitive: `ERROR` and `error` are the same), errors and critical against the window before, and the `topSystems` with the most alerts, then the most messages. The digest's own `system` never counts, nor do `excludeSystems` - on a cluster whose k8s-pitcher pitches every Event, `kubernetes` is a candidate.
- **Severity**: `error` when the window had a critical message; `warning` when it had more errors than the window before; `info` with errors, but not more; `success` otherwise.
- **Once**: each window is claimed in Redis (`SET scout:digest:<schedule>:<end> NX`, kept 8 days) before it is pitched, so several replicas and restarts pitch it once. A failed pitch releases the claim and is retried the next minute.
- **Late**: a digest is pitched up to 30 minutes (hourly) or 1 hour (daily) after its window ended; a scout that was down longer skips that window.
- **Comparison**: the window before is compared only while retention still holds it; with `retention.ttl: 48h` a daily digest compares yesterday with the day before.
- **Requires** the index to declare `timestamp_unix` NUMERIC (homerun-library v4.5.0+ writes it, scout v0.10.0+ declares it). An older index is reported in the log and by `/analytics/digest`, and no digest is pitched.

Preview any time with `GET /analytics/digest?schedule=hourly|daily` (see [API Usage](api-usage.md)).

### `spec.digest.*`

| Field | Env var | Default |
|---|---|---|
| `enabled` | `DIGEST_ENABLED` | `false` |
| `timezone` | `DIGEST_TIMEZONE` | `UTC` |
| `hourly` | `DIGEST_HOURLY` | `false` |
| `dailyAt` | `DIGEST_DAILY_AT` | (none) |
| `excludeSystems` | `DIGEST_EXCLUDE_SYSTEMS` (comma-separated) | (none) |
| `topSystems` | `DIGEST_TOP_SYSTEMS` | `3` |
| `system` | `DIGEST_SYSTEM` | `scout-digest` |

`enabled` and `hourly` in the CR only switch on: a CR without them leaves a digest enabled by env alone.
