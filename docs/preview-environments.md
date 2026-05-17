# Preview Environments

Every pull request opened against `main` can spin up an ephemeral, fully-deployed instance of scout on the `homerun2-dev` Kubernetes cluster — co-tenanted with omni-pitcher and redis-stack so reviewers can pitch real events and watch them surface in scout's analytics endpoints. The environment lives for as long as the PR is open and tears down automatically on merge or close.

This page covers how to use it, what each PR gets, the components that make it work, and how to troubleshoot.

## Quick start

1. Open a PR against `main`.
2. Add the `preview` label: `gh pr edit <num> --add-label preview`.
3. Wait 5–10 minutes for the image build, the kustomize-OCI push, and Argo's PullRequest generator poll (every 600s).
4. The preview-bot leaves a sticky comment on the PR with the URL.

Closing or merging the PR tears the namespace down automatically.

## What you get per PR

Each preview lives in its own namespace: `homerun2-scout-pr-<num>` on `homerun2-dev`. The namespace contains:

| Workload | Purpose |
|--|--|
| `homerun2-scout` | The system under test (this PR's commit) |
| `homerun2-omni-pitcher` (co-tenanted, pinned `v1.8.1`) | Produces events into the same Redis instance — `JSON.SET` populates the RediSearch index scout reads from |
| `redis-stack` | The data store (RediSearch + RedisJSON); persistence disabled (ephemeral) |
| `seed-test-events` (one-shot Job) | Posts a 5-event fixture to omni-pitcher right after the Deployment becomes Ready, so scout's analytics endpoints are non-empty on first poll |
| `verify-scout-analytics` (one-shot Job) | After seed completes + one aggregator tick, polls `scout/analytics/summary` and asserts `totalMessages >= 1` — fails the Job if the full pitcher → JSON.SET → RediSearch → scout chain didn't deliver |

Scout is reachable at: `https://scout-pr-<num>.homerun2-dev.sthings-vsphere.labul.sva.de`

The co-tenanted omni-pitcher is reachable in-cluster at the standard Service DNS. Its external HTTPRoute uses a distinct hostname (`omni-scout-pr-<num>.…`) so it doesn't collide with omni-pitcher's own per-PR previews or core-catcher's co-tenanted omni-pitcher (`omni-cc-pr-<num>.…`).

## Why the `preview` label gate

Without the label, every renovate / dependabot dep-bump PR would spawn a namespace. Two problems:

- Branches predating the build-pr workflow have no `pr-<num>-<sha>` image or kustomize artifacts published — half-empty namespaces with sync errors.
- Bots open dozens of PRs per week; the preview infrastructure isn't built for that scale.

Human-opened PRs opt in via the label. Bots don't apply it, so they're excluded by default. The Argo AppSet's PullRequest generator filters on `labels: [preview]`.

## The flow, end to end

```
git push (PR opens)
   ├─► comment-preview-url.yaml  ─►  sticky bot comment with URL
   ├─► build-scan-image.yaml     ─►  ko-built image at ghcr.io/.../homerun2-scout:pr-<num>-<sha>
   ├─► push-kustomize-pr.yaml    ─►  kustomize OCI at ghcr.io/.../homerun2-scout-kustomize:pr-<num>-<sha>
   └─► build-test.yaml + lint    ─►  CI gates

Argo PullRequest generator (poll every 600s)
   └─► detects PR with `preview` label
       └─► renders parent Application `homerun2-scout-pr-<num>` in argocd ns
           └─► chart emits child Applications targeting `homerun2-scout-pr-<num>` ns
               on the homerun2-dev cluster

Kyverno ClusterPolicies (auto-fire on namespace create)
   ├─► generate ResourceQuota + LimitRange
   ├─► generate 4 ExternalSecrets → ESO materializes Secrets from Vault
   ├─► generate seed Job (posts fixture after omni-pitcher Ready)
   └─► generate verify Job (polls scout /analytics/summary, asserts non-empty)

PR close
   ├─► AppSet drops the entry → finalizer cascade prunes child Apps + workloads
   ├─► cleanup-pr-artifacts.yaml deletes both ghcr.io packages
   └─► Kyverno ClusterCleanupPolicy reaps any empty namespace shell left behind
```

## The four PR-preview workflows in this repo

All four are in `.github/workflows/` and trigger on `pull_request` events targeting `main`.

| Workflow | Trigger | Output |
|--|--|--|
| `build-scan-image.yaml` | PR opened/updated | ko-built image tagged `pr-<num>-<sha>` + `pr-<num>` |
| `push-kustomize-pr.yaml` | PR opened/updated | kustomize OCI tagged `pr-<num>-<sha>` (renders `kcl/main.k` against `tests/kcl-deploy-profile.yaml`) |
| `comment-preview-url.yaml` | PR opened/reopened | Sticky comment with URL, namespace, ArgoCD link |
| `cleanup-pr-artifacts.yaml` | PR closed | Deletes both ghcr.io packages so version histories don't fill with PR debris |

Three of the four delegate to reusable workflows in `stuttgart-things/github-workflow-templates`. The `comment` one is inline because it adds a scout-specific note about the co-tenanted omni-pitcher.

## The Argo AppSet, briefly

Lives at `stuttgart-things/stuttgart-things` under `clusters/labul/vsphere/platform-sthings/argocd/homerun2-dev/scout-pr-preview-appset.yaml`. The shape:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: homerun2-scout-pr-preview
  namespace: argocd
spec:
  generators:
    - pullRequest:
        github:
          owner: stuttgart-things
          repo: homerun2-scout
          tokenRef: { secretName: homerun2-omni-pitcher-pat, key: token }
          labels: [preview]               # the gate
        requeueAfterSeconds: 600          # poll cadence
  template:
    metadata:
      name: 'homerun2-scout-pr-{{ .number }}'
      finalizers: [resources-finalizer.argocd.argoproj.io]   # cascade on prune
    spec:
      source:
        repoURL: https://github.com/stuttgart-things/argocd.git
        path: apps/homerun2/install
        helm:
          valuesObject:
            destination:
              name: homerun2-dev
              namespace: 'homerun2-scout-pr-{{ .number }}'
            scout:
              enabled: true
              version: 'pr-{{ .number }}-{{ .head_sha }}'
              hostname: 'scout-pr-{{ .number }}.homerun2-dev.sthings-vsphere.labul.sva.de'
              inlineHttpRoute: true       # Option B — see below
            omniPitcher:
              enabled: true
              version: v1.8.1
              hostname: 'omni-scout-pr-{{ .number }}.homerun2-dev.sthings-vsphere.labul.sva.de'
            redisStack:
              enabled: true
              persistence: { enabled: false }
              auth: { existingSecret: redis-stack-auth }
            # all other components off
            httpRoute:
              enabled: true
              gateway: { name: homerun2-dev-gateway, namespace: default }
      syncPolicy:
        automated: { prune: true, selfHeal: true }
        syncOptions: [CreateNamespace=true, ServerSideApply=true]
```

The AppSet renders one **parent** Argo `Application` per labelled PR. The parent's source is the `apps/homerun2/install` chart in the `stuttgart-things/argocd` catalog. The chart emits **child** Applications (one per enabled component: scout, omni-pitcher, redis-stack) on the homerun2-dev cluster.

`destination.name: homerun2-dev` (not a URL) means the chart targets the workload cluster by its registered Argo cluster name, so IP / DNS changes don't break manifests.

## The six cluster overlay manifests

Sit alongside the AppSet in `…/argocd/homerun2-dev/`:

| File | What it does |
|--|--|
| `scout-pr-preview-appset.yaml` | The ApplicationSet above |
| `homerun2-scout-preview-quota.yaml` | Kyverno `ClusterPolicy` → generates `ResourceQuota` + `LimitRange` in each PR namespace |
| `homerun2-scout-preview-secrets.yaml` | Kyverno `ClusterPolicy` → generates 4 `ExternalSecret`s; ESO pulls from Vault `homerun2-pr/data/preview-env` |
| `homerun2-scout-preview-seed-data.yaml` | Kyverno `ClusterPolicy` → generates the one-shot seed Job that posts the event fixture to omni-pitcher |
| `homerun2-scout-preview-verify.yaml` | Kyverno `ClusterPolicy` → generates the one-shot verify Job that polls `scout/analytics/summary` and fails if it's empty |
| `homerun2-scout-preview-sweep.yaml` | Kyverno `ClusterCleanupPolicy` → cron-reaps empty PR namespace shells |

These are deployed *once per cluster*. Per-PR, they fire automatically when the AppSet creates the namespace.

## The functional-test verify Job

What makes scout's preview env different from a typical web-service preview: it includes a **functional assertion** that the full data path works.

Flow inside `verify-scout-analytics`:

1. `curl /health` until 200 (60×5s).
2. Sleep 90s — enough for the seed Job to finish posting + one scout aggregator tick (default `scoutInterval: 60s`) to have run an `FT.AGGREGATE`.
3. `GET /analytics/summary` with the bearer token from `homerun2-scout-token`.
4. `sed`-extract `totalMessages`. Fail if `< 1`.

A clean preview run logs:

```
healthy after 1 attempts
sleeping 90s for seed Job + one aggregator tick...
polling /analytics/summary...
summary: {"totalMessages":28,"severityCounts":{...},"timeWindow":"1m0s",…}
totalMessages=28 minTotal=1
verify complete: scout sees 28 messages
```

If the verify Job fails, the seed Job ran but scout doesn't see the events — typically a regression in scout's `FT.CREATE` index schema (`internal/aggregator/index.go`) or its `FT.AGGREGATE` queries (`internal/aggregator/queries.go`). Inspect logs: `kubectl -n homerun2-scout-pr-<num> logs job/verify-scout-analytics`.

The `>= 1` assertion is intentionally tolerant: the known seed-Job JSON-splitting bug drops the last fixture event, and replay cycles inflate counts. Reviewers wanting a stricter floor bump `minTotalMessages` in the cluster-overlay Application's `valuesObject`.

## HTTPRoute: Option B (inline in the kustomize OCI)

The HTTPRoute exposing scout externally is rendered by `kcl/httproute.k` and ships **inside the kustomize OCI**, alongside the Service. They land in the same kustomize apply, eliminating the cross-Application race that previously let Cilium's gateway controller stamp a sticky `BackendNotFound` (tracked under [stuttgart-things/argocd#116](https://github.com/stuttgart-things/argocd/issues/116)). Three places have to agree:

| Repo | Setting |
|--|--|
| `homerun2-scout` (this repo) | `tests/kcl-deploy-profile.yaml` → `config.httpRouteEnabled: true` |
| `stuttgart-things/argocd` | `apps/homerun2/install` → `scout.inlineHttpRoute` flag patches the rendered HTTPRoute's parentRef + hostname per env, and excludes scout from the standalone httproute Application |
| `stuttgart-things/stuttgart-things` | Set `scout.inlineHttpRoute: true` in the AppSet's `valuesObject` |

With all three set, `HTTPRoute/homerun2-scout` lands `ResolvedRefs: True` on first reconcile. No manual `kubectl annotate httproute reconcile-bump=$(date +%s) --overwrite` required.

## Lifecycle

| Event | Result |
|--|--|
| PR opened with `preview` label | Sticky bot comment posted; CI builds image + kustomize OCI; AppSet picks it up within 600s; namespace + workloads spin up |
| PR updated (new commit) | Image + kustomize OCI rebuilt with new `<sha>`; AppSet detects the head-SHA change; rolling update of Deployments |
| PR `preview` label removed | AppSet drops the entry; finalizer prune cascades teardown |
| PR closed (merged or rejected) | AppSet drops the entry → teardown; `cleanup-pr-artifacts.yaml` deletes ghcr.io packages |

The `resources-finalizer.argocd.argoproj.io` finalizer on the parent Application is critical — without it, Argo would delete the parent instantly when the AppSet drops it, orphaning child Apps + workload pods. With it, Argo runs prune on every managed resource first.

## Troubleshooting

| Symptom | Likely cause | Fix |
|--|--|--|
| No bot comment, no namespace | `preview` label missing | `gh pr edit <num> --add-label preview` |
| Bot comment present, namespace never appears | AppSet hasn't polled yet | Wait up to 10 min, or `kubectl -n argocd annotate appset homerun2-scout-pr-preview argocd.argoproj.io/refresh=hard` |
| Parent Application sync error: `failed to load: oci pull` | Image / kustomize OCI build still running or failed | Check the PR's Actions tab — `build-pr` and `push-kustomize` must both be green |
| Pods stuck `ImagePullBackOff` | ghcr.io tag not yet pushed (CI still running) or PR closed (cleanup workflow already ran) | Wait for build / reopen the PR |
| Pods CrashLoopBackOff with `WRONGPASS` | ESO hasn't materialized `redis-stack-auth` Secret yet | Check `kubectl -n homerun2-scout-pr-<num> get externalsecret`; refresh if not Ready |
| HTTPRoute `ResolvedRefs: False` | Service didn't land before HTTPRoute (pre-Option-B environments only) | Should not happen now; if it does: `kubectl annotate httproute homerun2-scout reconcile-bump=$(date +%s) --overwrite -n homerun2-scout-pr-<num>` and file an issue |
| `/analytics/summary` returns `{"totalMessages":0,…}` | Seed Job hasn't run, omni-pitcher's `JSON.SET` path is broken, or scout's RediSearch index isn't picking up keys | `kubectl -n homerun2-scout-pr-<num> get jobs` — both `seed-test-events` and `verify-scout-analytics` should be `Complete`. If verify failed, check its logs for the captured summary response. |
| Verify Job fails with `parse error` | `/analytics/summary` returned non-JSON or an error body | Likely scout's pod is unhealthy. `kubectl logs deploy/homerun2-scout` |
| Namespace stuck Terminating after PR close | Finalizer on a CRD instance | `kubectl get all,externalsecret -n homerun2-scout-pr-<num>` to find the blocker |

## See also

- [stuttgart-things/argocd `apps/homerun2`](https://github.com/stuttgart-things/argocd/tree/main/apps/homerun2) — the install chart + Kyverno policy charts (including `preview-seed-data` + `preview-scout-verify`)
- [stuttgart-things/homerun2-omni-pitcher#116](https://github.com/stuttgart-things/homerun2-omni-pitcher/issues/116) — the umbrella rollout issue tracking all 8 components
- [stuttgart-things/argocd#116](https://github.com/stuttgart-things/argocd/issues/116) — the HTTPRoute creation-order race writeup that motivated Option B
- [stuttgart-things/github-workflow-templates](https://github.com/stuttgart-things/github-workflow-templates) — the three reusable PR-preview workflows this repo delegates to
