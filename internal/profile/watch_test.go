package profile

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// seqLoader returns its steps in order, then keeps returning the last one.
type seqLoader struct {
	mu    sync.Mutex
	steps []loadStep
	calls int
}

type loadStep struct {
	profile *ScoutProfile
	err     error
}

func (l *seqLoader) Load(context.Context, string, string) (*ScoutProfile, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	i := min(l.calls, len(l.steps)-1)
	l.calls++
	return l.steps[i].profile, l.steps[i].err
}

// notFound is the error KubernetesLoader returns for a missing CR: the API's
// NotFound, wrapped.
func notFound() error {
	return fmt.Errorf("get ScoutProfile homerun2/default: %w",
		apierrors.NewNotFound(schema.GroupResource{Group: "homerun2.stuttgart-things.com", Resource: "scoutprofiles"}, "default"))
}

func profileWithDaily(at string) *ScoutProfile {
	return &ScoutProfile{Retention: RetentionSpec{Enabled: true}, Digest: DigestSpec{Enabled: true, DailyAt: at}}
}

// watch runs Watch until it reports a change or timeout passes, and returns the
// reasons it reported.
func watch(t *testing.T, loader ProfileLoader, loaded *ScoutProfile, timeout time.Duration) []string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	var mu sync.Mutex
	var reasons []string
	Watch(ctx, loader, "homerun2", "default", loaded, 5*time.Millisecond, func(reason string) {
		mu.Lock()
		defer mu.Unlock()
		reasons = append(reasons, reason)
	})
	return reasons
}

func TestWatch(t *testing.T) {
	top := 3
	cases := []struct {
		name   string
		loaded *ScoutProfile
		steps  []loadStep
		want   []string
	}{
		{name: "unchanged", loaded: profileWithDaily("07:00"),
			steps: []loadStep{{profile: profileWithDaily("07:00")}}},
		{name: "changed spec", loaded: profileWithDaily("07:00"),
			steps: []loadStep{{profile: profileWithDaily("07:00")}, {profile: profileWithDaily("08:00")}},
			want:  []string{"ScoutProfile changed"}},
		{name: "changed pointer field", loaded: profileWithDaily("07:00"),
			steps: []loadStep{{profile: &ScoutProfile{Retention: RetentionSpec{Enabled: true}, Digest: DigestSpec{Enabled: true, DailyAt: "07:00", TopSystems: &top}}}},
			want:  []string{"ScoutProfile changed"}},
		{name: "created after startup", loaded: nil,
			steps: []loadStep{{err: notFound()}, {profile: profileWithDaily("07:00")}},
			want:  []string{"ScoutProfile created"}},
		{name: "deleted", loaded: profileWithDaily("07:00"),
			steps: []loadStep{{err: notFound()}},
			want:  []string{"ScoutProfile deleted"}},
		{name: "still missing", loaded: nil,
			steps: []loadStep{{err: notFound()}}},
		{name: "API unreachable, then unchanged", loaded: profileWithDaily("07:00"),
			steps: []loadStep{{err: errors.New("connection refused")}, {profile: profileWithDaily("07:00")}}},
		{name: "forbidden is not missing", loaded: profileWithDaily("07:00"),
			steps: []loadStep{{err: apierrors.NewForbidden(schema.GroupResource{Resource: "scoutprofiles"}, "default", errors.New("rbac"))}}},
		{name: "API unreachable, then changed", loaded: profileWithDaily("07:00"),
			steps: []loadStep{{err: errors.New("timeout")}, {err: errors.New("timeout")}, {profile: profileWithDaily("09:00")}},
			want:  []string{"ScoutProfile changed"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			loader := &seqLoader{steps: tc.steps}
			got := watch(t, loader, tc.loaded, 150*time.Millisecond)
			if fmt.Sprint(got) != fmt.Sprint(tc.want) {
				t.Errorf("reasons = %q, want %q", got, tc.want)
			}
			if loader.calls < 2 && tc.want == nil {
				t.Errorf("loader read %d times in 150ms; the watch did not poll", loader.calls)
			}
		})
	}
}

func TestWatchReportsOnceAndStops(t *testing.T) {
	loader := &seqLoader{steps: []loadStep{{profile: profileWithDaily("08:00")}}}
	start := time.Now()
	got := watch(t, loader, profileWithDaily("07:00"), time.Second)
	if len(got) != 1 {
		t.Errorf("reasons = %q, want exactly one", got)
	}
	if time.Since(start) > 500*time.Millisecond {
		t.Error("Watch kept running after reporting the change")
	}
}

func TestWatchStopsWithTheContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		Watch(ctx, &seqLoader{steps: []loadStep{{profile: profileWithDaily("07:00")}}}, "homerun2", "default",
			profileWithDaily("07:00"), time.Millisecond, func(string) { t.Error("unexpected change") })
	}()
	time.Sleep(20 * time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Watch did not return after the context was cancelled")
	}
}
