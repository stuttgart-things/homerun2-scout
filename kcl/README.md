# homerun2-scout / KCL Deployment

KCL-based Kubernetes manifests for homerun2-scout. Renders Namespace, ServiceAccount, ConfigMap, Secrets, Deployment, Service, and optionally Ingress or HTTPRoute.

## Render Manifests

### Via Dagger (recommended)

```bash
# render with a profile file
dagger call -m github.com/stuttgart-things/dagger/kcl@v0.82.0 run \
  --source kcl \
  --parameters-file tests/kcl-movie-scripts-profile.yaml \
  export --path /tmp/rendered-homerun2-scout.yaml

# render with inline parameters
dagger call -m github.com/stuttgart-things/dagger/kcl@v0.82.0 run \
  --source kcl \
  --parameters 'config.image=ghcr.io/stuttgart-things/homerun2-scout:v0.1.0,config.namespace=homerun2' \
  export --path /tmp/rendered-homerun2-scout.yaml
```

### Via kcl CLI

```bash
kcl run kcl/main.k \
  -D 'config.image=ghcr.io/stuttgart-things/homerun2-scout:v0.1.0' \
  -D 'config.namespace=homerun2'
```

### Via Taskfile

```bash
task render-manifests-quick
```

## Deploy to Cluster

```bash
# render + apply
dagger call -m github.com/stuttgart-things/dagger/kcl@v0.82.0 run \
  --source kcl \
  --parameters-file tests/kcl-movie-scripts-profile.yaml \
  export --path /tmp/rendered-homerun2-scout.yaml

KUBECONFIG=~/.kube/movie-scripts kubectl apply -f /tmp/rendered-homerun2-scout.yaml
```

## Profile Parameters

| Parameter | Default | Description |
|---|---|---|
| `config.image` | `ghcr.io/stuttgart-things/homerun2-scout:latest` | Container image |
| `config.namespace` | `homerun2` | Target namespace |
| `config.replicas` | `1` | Replica count |
| `config.serviceType` | `ClusterIP` | Service type |
| `config.servicePort` | `80` | Service port |
| `config.containerPort` | `8080` | Container port |
| `config.cpuRequest` | `100m` | CPU request |
| `config.cpuLimit` | `500m` | CPU limit |
| `config.memoryRequest` | `128Mi` | Memory request |
| `config.memoryLimit` | `512Mi` | Memory limit |
| `config.redisAddr` | `redis-stack.homerun2.svc.cluster.local` | Redis host |
| `config.redisPort` | `6379` | Redis port |
| `config.redisearchIndex` | `messages` | RediSearch index name |
| `config.scoutInterval` | `60s` | Aggregation interval |
| `config.authToken` | *(empty)* | Bearer auth token (creates Secret if set) |
| `config.redisPassword` | *(empty)* | Redis password (creates Secret if set) |
| `config.ingressEnabled` | `false` | Enable Ingress |
| `config.ingressClassName` | `nginx` | Ingress class |
| `config.ingressHost` | `homerun2-scout.example.com` | Ingress hostname |
| `config.ingressTlsEnabled` | `false` | Enable Ingress TLS |
| `config.ingressAnnotations` | `{}` | Extra Ingress annotations |
| `config.httpRouteEnabled` | `false` | Enable HTTPRoute (Gateway API) |
| `config.httpRouteParentRefName` | *(empty)* | Gateway name |
| `config.httpRouteParentRefNamespace` | *(empty)* | Gateway namespace |
| `config.httpRouteHostname` | *(empty)* | HTTPRoute hostname |

## Example Profiles

### movie-scripts cluster (HTTPRoute + redis-stack)

```yaml
---
config.image: ghcr.io/stuttgart-things/homerun2-scout:v0.1.0
config.namespace: homerun2
config.ingressEnabled: false
config.httpRouteEnabled: true
config.httpRouteParentRefName: movie-scripts2-gateway
config.httpRouteParentRefNamespace: default
config.httpRouteHostname: homerun2-scout.movie-scripts2.sthings-vsphere.labul.sva.de
config.redisAddr: redis-stack.redis-stack.svc.cluster.local
config.redisPort: "6379"
config.redisearchIndex: messages
config.scoutInterval: "60s"
config.authToken: <your-token>
config.redisPassword: <your-password>
```

### dev cluster (Ingress + local Redis)

```yaml
---
config.image: ghcr.io/stuttgart-things/homerun2-scout:latest
config.namespace: homerun2
config.ingressEnabled: true
config.ingressHost: homerun2-scout.example.com
config.ingressTlsEnabled: true
config.ingressAnnotations:
  cert-manager.io/cluster-issuer: cluster-issuer-approle
config.redisAddr: redis-stack.homerun2.svc.cluster.local
config.redisPort: "6379"
config.redisearchIndex: messages
config.scoutInterval: "60s"
config.authToken: changeme
config.redisPassword: changeme
```
