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
    enabled: true           # Enable periodic RediSearch index cleanup
    ttl: 168h               # Max age of entries to keep (Go duration)
  alerting:
    pitcherURL: https://...  # omni-pitcher /pitch endpoint
    pitcherToken: ""         # Bearer token — prefer ALERT_PITCHER_TOKEN env var
    errorThreshold: 50       # Error count that triggers a meta-alert
    criticalThreshold: 10    # Critical count that triggers a meta-alert
    cooldown: 5m             # Minimum time between alerts (Go duration)
```

## How It Works

1. At startup, if `SCOUT_PROFILE_NAME` is set, the scout reads the named CR from the pod's namespace via the Kubernetes API
2. Non-empty fields in the CR **override** the corresponding env var values in config
3. If the CR is missing, unreachable, or `SCOUT_PROFILE_NAME` is empty, the scout starts normally with env var defaults — no crash

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
    pitcherURL: https://homerun2-omni-pitcher.movie-scripts2.sthings-vsphere.labul.sva.de/pitch
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
export ALERT_PITCHER_URL=http://localhost:8081/pitch
export ALERT_ERROR_THRESHOLD=50
export ALERT_CRITICAL_THRESHOLD=10
export SCOUT_RETENTION_TTL=168h
go run .
```

## Field Reference

### `spec.scoutInterval`

Go duration string (e.g. `30s`, `2m`, `1h`). Overrides `SCOUT_INTERVAL` env var.

### `spec.retention.enabled`

Boolean. Enables periodic cleanup of RediSearch entries older than `ttl`.

### `spec.retention.ttl`

Go duration string (e.g. `168h` = 7 days). Overrides `SCOUT_RETENTION_TTL` env var.

### `spec.alerting.pitcherURL`

Full URL of the omni-pitcher `/pitch` endpoint. Overrides `ALERT_PITCHER_URL` env var.

### `spec.alerting.pitcherToken`

Bearer token for omni-pitcher. It is recommended to leave this empty in the CR and use the `ALERT_PITCHER_TOKEN` env var (sourced from a Kubernetes Secret) instead.

### `spec.alerting.errorThreshold`

Integer. Number of `error`-severity messages in a single aggregation cycle that triggers a meta-alert. Overrides `ALERT_ERROR_THRESHOLD` env var.

### `spec.alerting.criticalThreshold`

Integer. Number of `critical`-severity messages that triggers a meta-alert. Overrides `ALERT_CRITICAL_THRESHOLD` env var.

### `spec.alerting.cooldown`

Go duration string. Minimum time between successive alerts to avoid alert storms. Overrides `ALERT_COOLDOWN` env var.
