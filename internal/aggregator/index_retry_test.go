package aggregator

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestIsMissingIndexError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"redis stack 7.2", errors.New("messages: no such index"), true},
		{"older redisearch", errors.New("Unknown index name"), true},
		{"connection refused", errors.New("dial tcp 10.43.0.1:6379: connect: connection refused"), false},
		{"timeout", context.DeadlineExceeded, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isMissingIndexError(tc.err); got != tc.want {
				t.Errorf("isMissingIndexError(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

func TestEnsureIndexIfMissing_RetriesUntilItSucceeds(t *testing.T) {
	agg := New(nil, "messages", time.Minute)
	calls := 0
	agg.ensureIndex = func(context.Context) error {
		calls++
		if calls < 3 {
			return errors.New("dial tcp: connect: connection refused")
		}
		return nil
	}

	for i := 0; i < 5; i++ {
		agg.ensureIndexIfMissing(context.Background())
	}

	if calls != 3 {
		t.Errorf("ensureIndex calls = %d, want 3 (retried while failing, then no more once the index exists)", calls)
	}
	if !agg.indexReady.Load() {
		t.Error("indexReady = false after a successful ensureIndex")
	}
}

func TestNoteQueryError_MissingIndexTriggersRecreation(t *testing.T) {
	agg := New(nil, "messages", time.Minute)
	calls := 0
	agg.ensureIndex = func(context.Context) error { calls++; return nil }

	agg.ensureIndexIfMissing(context.Background())
	agg.noteQueryError(errors.New("messages: no such index"))
	if agg.indexReady.Load() {
		t.Fatal("indexReady still true after a query reported the index missing")
	}
	agg.ensureIndexIfMissing(context.Background())

	if calls != 2 {
		t.Errorf("ensureIndex calls = %d, want 2 (initial + re-creation)", calls)
	}
}

func TestNoteQueryError_OtherErrorsKeepTheIndex(t *testing.T) {
	agg := New(nil, "messages", time.Minute)
	agg.indexReady.Store(true)

	agg.noteQueryError(errors.New("i/o timeout"))
	agg.noteQueryError(nil)

	if !agg.indexReady.Load() {
		t.Error("indexReady reset by an error that is not a missing index")
	}
}
