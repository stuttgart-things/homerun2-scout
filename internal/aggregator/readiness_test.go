package aggregator

import (
	"context"
	"errors"
	"testing"
	"time"
)

func newTestAggregator(interval time.Duration, now *time.Time) *Aggregator {
	a := New(nil, "messages", interval)
	a.now = func() time.Time { return *now }
	a.ensureIndex = func(context.Context) error { return nil }
	return a
}

func TestReadiness_NotReadyBeforeFirstCycle(t *testing.T) {
	now := time.Date(2026, 9, 12, 10, 0, 0, 0, time.UTC)
	a := newTestAggregator(time.Minute, &now)
	ready, resp := a.Readiness()
	if ready {
		t.Fatalf("ready before any cycle: %+v", resp)
	}
	if resp.Reason == "" {
		t.Error("not ready without a reason")
	}
}

func TestReadiness_ReadyAfterSuccessfulCycle(t *testing.T) {
	now := time.Date(2026, 9, 12, 10, 0, 0, 0, time.UTC)
	a := newTestAggregator(time.Minute, &now)
	a.cycleErrors.Store(0)
	a.ensureIndexIfMissing(context.Background())
	a.recordCycle()

	ready, resp := a.Readiness()
	if !ready {
		t.Fatalf("not ready after a successful cycle: %+v", resp)
	}
	if resp.LastSuccess == "" || resp.ConsecutiveFailures != 0 {
		t.Errorf("unexpected response %+v", resp)
	}
}

func TestReadiness_OneFailedCycleStaysReady(t *testing.T) {
	now := time.Date(2026, 9, 12, 10, 0, 0, 0, time.UTC)
	a := newTestAggregator(time.Minute, &now)
	a.ensureIndexIfMissing(context.Background())
	a.recordCycle()

	now = now.Add(time.Minute)
	a.cycleErrors.Store(0)
	a.noteQueryError(errors.New("i/o timeout"))
	a.recordCycle()

	ready, resp := a.Readiness()
	if !ready {
		t.Fatalf("one failed cycle within staleAfter made scout unready: %+v", resp)
	}
	if resp.ConsecutiveFailures != 1 || resp.LastError != "i/o timeout" {
		t.Errorf("failure not reported: %+v", resp)
	}

	now = now.Add(time.Minute)
	a.cycleErrors.Store(0)
	a.recordCycle()
	if _, resp := a.Readiness(); resp.LastError != "" || resp.ConsecutiveFailures != 0 {
		t.Errorf("a successful cycle should clear the last error and the failure count: %+v", resp)
	}
}

func TestReadiness_StaleAfterThreeIntervals(t *testing.T) {
	now := time.Date(2026, 9, 12, 10, 0, 0, 0, time.UTC)
	a := newTestAggregator(time.Minute, &now)
	a.ensureIndexIfMissing(context.Background())
	a.recordCycle()

	for i := 0; i < 4; i++ {
		now = now.Add(time.Minute)
		a.cycleErrors.Store(0)
		a.noteQueryError(errors.New("dial tcp: connection refused"))
		a.recordCycle()
	}

	ready, resp := a.Readiness()
	if ready {
		t.Fatalf("still ready 4 intervals after the last success: %+v", resp)
	}
	if resp.ConsecutiveFailures != 4 {
		t.Errorf("ConsecutiveFailures = %d, want 4", resp.ConsecutiveFailures)
	}
}

func TestReadiness_MissingIndexIsNotReadyAtOnce(t *testing.T) {
	now := time.Date(2026, 9, 12, 10, 0, 0, 0, time.UTC)
	a := newTestAggregator(time.Minute, &now)
	a.ensureIndexIfMissing(context.Background())
	a.recordCycle()

	a.noteQueryError(errors.New("messages: no such index"))

	if ready, resp := a.Readiness(); ready || resp.IndexReady {
		t.Fatalf("ready with the index gone: %+v", resp)
	}
}

func TestReadiness_FailedIndexCreationIsAFailedCycle(t *testing.T) {
	now := time.Date(2026, 9, 12, 10, 0, 0, 0, time.UTC)
	a := newTestAggregator(time.Minute, &now)
	a.ensureIndex = func(context.Context) error { return errors.New("dial tcp: connection refused") }

	a.cycleErrors.Store(0)
	a.ensureIndexIfMissing(context.Background())
	a.recordCycle()

	ready, resp := a.Readiness()
	if ready || resp.ConsecutiveFailures != 1 || resp.LastError == "" {
		t.Fatalf("failed index creation not reported: ready=%v %+v", ready, resp)
	}
}
