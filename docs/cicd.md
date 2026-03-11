# CI/CD

## GitHub Actions Workflows

| Workflow | Trigger | Description |
|----------|---------|-------------|
| `build-test.yaml` | PR / push to main | Dagger lint + build + test |
| `build-scan-image.yaml` | Push to main | ko build + Trivy scan |
| `release.yaml` | After image build / manual | Semantic release + stage image + push kustomize OCI |
| `lint-repo.yaml` | PR / push to main | Repository linting |
| `pages.yaml` | After release / manual | Deploy MkDocs to GitHub Pages |

## Dagger Functions

The `dagger/` module provides:

| Function | Description |
|----------|-------------|
| `Lint` | Go linting via golangci-lint |
| `Build` | Build Go binary |
| `BuildImage` | Build container image with ko |
| `ScanImage` | Trivy vulnerability scan |
| `BuildAndTestBinary` | Build + Redis integration test |

## Taskfile

Common tasks available via `task`:

```bash
task lint                  # Run golangci-lint
task build-test-binary     # Build and test with Redis via Dagger
task build-output-binary   # Build Go binary
task build-scan-image-ko   # Build + scan with ko
task render-manifests-quick # Render KCL manifests
task trigger-release       # Trigger release workflow
```

## Release Process

Releases are automated via semantic-release:

1. Push to `main` triggers build + image workflow
2. On success, release workflow runs semantic-release
3. If releasable commits exist, a new version tag is created
4. Container image is staged from `:main` to `:vX.Y.Z`
5. Kustomize base is pushed as OCI artifact to GHCR
