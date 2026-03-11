# CLAUDE.md

## Project

homerun2-scout — Go microservice that periodically analyzes messages indexed in RediSearch and exposes aggregated analytics via a REST API.

## Tech Stack

- **Language**: Go 1.25+
- **HTTP**: stdlib `net/http` (no framework)
- **Data**: RediSearch `FT.AGGREGATE` queries via `go-redis/v9`
- **Build**: ko (`.ko.yaml`), no Dockerfile
- **CI**: Dagger modules (`dagger/main.go`), Taskfile
- **Deploy**: KCL manifests (`kcl/`), Kustomize, Kubernetes
- **Infra**: GitHub Actions, semantic-release, renovate

## Git Workflow

**Branch-per-issue with PR and merge.** Every change gets its own branch, PR, and merge to main.

### Branch naming

- `fix/<issue-number>-<short-description>` for bugs
- `feat/<issue-number>-<short-description>` for features
- `test/<issue-number>-<short-description>` for test-only changes

### Commit messages

- Use conventional commits: `fix:`, `feat:`, `test:`, `chore:`, `docs:`
- End with `Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>` when Claude authored
- Include `Closes #<issue-number>` to auto-close issues

## Code Conventions

- No Dockerfile — use ko for image builds
- Config via environment variables, loaded once at startup
- Auth via Bearer token middleware (`AUTH_TOKEN` env var)
- Tests: `go test ./...` — unit tests must not require Redis
- KCL is the source of truth for Kubernetes manifests

## Key Paths

- `main.go` — entrypoint, routing, aggregator start, graceful shutdown
- `internal/aggregator/` — periodic FT.AGGREGATE queries, result caching
- `internal/handlers/` — HTTP handlers (analytics, health)
- `internal/middleware/` — auth middleware
- `internal/config/` — env-based config loading
- `internal/models/` — analytics response structs
- `dagger/main.go` — CI functions (Lint, Build, BuildImage, ScanImage, BuildAndTestBinary)
- `kcl/` — Kubernetes manifests
- `Taskfile.yaml` — task runner for build/test/deploy/release

## Testing

```bash
# Unit tests (no Redis needed)
go test ./...

# Integration test via Dagger (spins up Redis)
task build-test-binary

# Lint
task lint

# Build + scan image
task build-scan-image-ko

# Render KCL manifests
task render-manifests-quick
```
