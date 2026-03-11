// Dagger CI module for homerun2-scout
//
// Provides build, lint, test, image build, and security scanning functions.

package main

import (
	"context"
	"dagger/dagger/internal/dagger"
	"fmt"
)

type Dagger struct{}

// Lint runs golangci-lint on the source code
func (m *Dagger) Lint(
	ctx context.Context,
	src *dagger.Directory,
	// +optional
	// +default="500s"
	timeout string,
) *dagger.Container {
	return dag.Go().Lint(src, dagger.GoLintOpts{
		Timeout: timeout,
	})
}

// Build compiles the Go binary
func (m *Dagger) Build(
	ctx context.Context,
	src *dagger.Directory,
	// +optional
	// +default="main"
	binName string,
	// +optional
	// +default=""
	ldflags string,
	// +optional
	// +default="1.25.4"
	goVersion string,
	// +optional
	// +default="linux"
	os string,
	// +optional
	// +default="amd64"
	arch string,
) *dagger.Directory {
	return dag.Go().BuildBinary(src, dagger.GoBuildBinaryOpts{
		GoVersion:  goVersion,
		Os:         os,
		Arch:       arch,
		BinName:    binName,
		Ldflags:    ldflags,
		GoMainFile: "main.go",
	})
}

// BuildImage builds a container image using ko and optionally pushes it
func (m *Dagger) BuildImage(
	ctx context.Context,
	src *dagger.Directory,
	// +optional
	// +default="ko.local/homerun2-scout"
	repo string,
	// +optional
	// +default="false"
	push string,
) (string, error) {
	return dag.Go().KoBuild(ctx, src, dagger.GoKoBuildOpts{
		Repo: repo,
		Push: push,
	})
}

// ScanImage scans a container image for vulnerabilities using Trivy
func (m *Dagger) ScanImage(
	ctx context.Context,
	imageRef string,
	// +optional
	// +default="HIGH,CRITICAL"
	severity string,
) *dagger.File {
	return dag.Trivy().ScanImage(imageRef, dagger.TrivyScanImageOpts{
		Severity: severity,
	})
}

// BuildAndTestBinary builds the binary and runs integration tests with Redis
func (m *Dagger) BuildAndTestBinary(
	ctx context.Context,
	source *dagger.Directory,
	// +optional
	// +default="1.25.4"
	goVersion string,
	// +optional
	// +default="linux"
	os string,
	// +optional
	// +default="amd64"
	arch string,
	// +optional
	// +default="main.go"
	goMainFile string,
	// +optional
	// +default="main"
	binName string,
	// +optional
	// +default=""
	ldflags string,
	// +optional
	// +default="."
	packageName string,
	// +optional
	// +default="./..."
	testPath string,
	// +optional
	// +default="8080"
	port int,
) (*dagger.File, error) {

	binDir := dag.Go().BuildBinary(
		source,
		dagger.GoBuildBinaryOpts{
			GoVersion:   goVersion,
			Os:          os,
			Arch:        arch,
			GoMainFile:  goMainFile,
			BinName:     binName,
			Ldflags:     ldflags,
			PackageName: packageName,
		})

	redisService := dag.Homerun().RedisService(dagger.HomerunRedisServiceOpts{
		Version:  "7.2.0-v18",
		Password: "",
	})

	testCmd := fmt.Sprintf(`
exec > /app/test-output.log 2>&1
set -e

echo "=== Starting binary ==="
./%s &
BIN_PID=$!
sleep 3

echo ""
echo "=== Testing health endpoint ==="
curl -f -v http://localhost:%d/health || {
  echo "Health check failed!"
  kill $BIN_PID 2>/dev/null || true
  exit 1
}

echo ""
echo "=== Testing analytics/summary endpoint ==="
curl -f -v http://localhost:%d/analytics/summary \
  -H "Authorization: Bearer test-token-12345" || {
  echo "Analytics summary endpoint failed!"
  kill $BIN_PID 2>/dev/null || true
  exit 1
}

echo ""
echo "=== All tests passed! ==="
kill $BIN_PID 2>/dev/null || true
exit 0
`, binName, port, port)

	result := dag.Container().
		From("alpine:latest").
		WithExec([]string{"apk", "add", "--no-cache", "curl"}, dagger.ContainerWithExecOpts{}).
		WithDirectory("/app", binDir).
		WithWorkdir("/app").
		WithServiceBinding("redis", redisService).
		WithEnvVariable("REDIS_ADDR", "redis").
		WithEnvVariable("REDIS_PORT", "6379").
		WithEnvVariable("REDISEARCH_INDEX", "messages").
		WithEnvVariable("AUTH_TOKEN", "test-token-12345").
		WithEnvVariable("SCOUT_INTERVAL", "30s").
		WithExec([]string{"sh", "-c", testCmd}, dagger.ContainerWithExecOpts{})

	_, err := result.Sync(ctx)
	if err != nil {
		testLog := result.File("/app/test-output.log")
		return testLog, fmt.Errorf("tests failed - check test-output.log for details: %w", err)
	}

	testLog := result.File("/app/test-output.log")
	return testLog, nil
}
