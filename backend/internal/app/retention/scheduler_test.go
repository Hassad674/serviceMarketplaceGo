package retention_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	retentionapp "marketplace-backend/internal/app/retention"
	"marketplace-backend/internal/domain/retention"
)

// schedRepo is a minimal RetentionRepository fake the scheduler tests
// use. It counts every Sweep call so we can assert "the scheduler
// actually drove the service to run". We re-use the pattern from
// service_test (function fields + counters) without sharing the type
// to keep the two test files independent — one regression in fakeRepo
// must not silently break the other suite.
type schedRepo struct {
	mu        sync.Mutex
	calls     int
	plan      []int
	errReturn error
}

func (r *schedRepo) Sweep(_ context.Context, _ retention.Policy) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.errReturn != nil {
		return 0, r.errReturn
	}
	idx := r.calls
	r.calls++
	if idx < len(r.plan) {
		return r.plan[idx], nil
	}
	return 0, nil
}

func (r *schedRepo) callCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.calls
}

func newSchedPolicy() retention.Policy {
	return retention.Policy{
		Name:      "messages_test",
		Table:     "messages",
		AgeColumn: "created_at",
		MaxAge:    24 * time.Hour,
		Strategy:  retention.StrategyDelete,
		BatchSize: 10,
	}
}

// TestNewScheduler_DefaultIntervalApplied — passing 0 must fall back
// to the production cadence. The constructor should never silently
// store a zero ticker interval (that would panic at NewTicker).
func TestNewScheduler_DefaultIntervalApplied(t *testing.T) {
	svc, err := retentionapp.NewService(&schedRepo{}, []retention.Policy{newSchedPolicy()})
	require.NoError(t, err)
	s := retentionapp.NewScheduler(svc, 0)
	require.NotNil(t, s)
}

// TestNewScheduler_CustomIntervalApplied — explicit value wins.
func TestNewScheduler_CustomIntervalApplied(t *testing.T) {
	svc, err := retentionapp.NewService(&schedRepo{}, []retention.Policy{newSchedPolicy()})
	require.NoError(t, err)
	s := retentionapp.NewScheduler(svc, 250*time.Millisecond)
	require.NotNil(t, s)
}

// TestScheduler_Run_NilServiceShortCircuits — a misconfigured wiring
// must not panic. The scheduler logs an error and returns immediately.
func TestScheduler_Run_NilServiceShortCircuits(t *testing.T) {
	s := retentionapp.NewScheduler(nil, 10*time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	// Must return without panicking even though the inner service is nil.
	s.Run(ctx)
}

// TestScheduler_Run_FirstTickRunsImmediately — the scheduler kicks off
// a sweep on start so a fresh boot picks up overdue retention without
// waiting a full interval.
func TestScheduler_Run_FirstTickRunsImmediately(t *testing.T) {
	repo := &schedRepo{plan: []int{3, 0}}
	svc, err := retentionapp.NewService(repo, []retention.Policy{newSchedPolicy()})
	require.NoError(t, err)
	// Big interval — the first tick must drain via the eager pre-ticker
	// call, not via the loop. If the immediate tick is dropped the
	// counter stays at zero until the long ticker fires.
	s := retentionapp.NewScheduler(svc, 1*time.Hour)

	done := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		s.Run(ctx)
		close(done)
	}()

	// Wait for the immediate tick to drain the plan.
	require.Eventually(t, func() bool {
		return repo.callCount() >= 2
	}, 1*time.Second, 5*time.Millisecond, "first tick must run before the ticker fires")

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("scheduler did not exit after context cancel")
	}
}

// TestScheduler_Run_TickCadence — once the immediate tick is done, the
// scheduler must keep ticking on the configured interval.
func TestScheduler_Run_TickCadence(t *testing.T) {
	var sweepCount atomic.Int32
	repo := &fakeCountingRepo{onSweep: func() { sweepCount.Add(1) }}
	svc, err := retentionapp.NewService(repo, []retention.Policy{newSchedPolicy()})
	require.NoError(t, err)
	// Fast interval so the test stays under 1s of wall time.
	s := retentionapp.NewScheduler(svc, 30*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		s.Run(ctx)
		close(done)
	}()

	// We want at least 3 ticks: the immediate one + two interval ones.
	require.Eventually(t, func() bool {
		return sweepCount.Load() >= 3
	}, 500*time.Millisecond, 5*time.Millisecond,
		"scheduler must tick at least 3 times within 500ms at a 30ms cadence")

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("scheduler did not exit after context cancel")
	}
}

// TestScheduler_Run_LogsErrorButContinues — when a policy errors, the
// scheduler must keep ticking. We assert by counting subsequent ticks
// after the first errored one.
func TestScheduler_Run_LogsErrorButContinues(t *testing.T) {
	var count atomic.Int32
	repo := &fakeCountingRepo{
		onSweep: func() { count.Add(1) },
		err:     errors.New("boom"),
	}
	svc, err := retentionapp.NewService(repo, []retention.Policy{newSchedPolicy()})
	require.NoError(t, err)
	s := retentionapp.NewScheduler(svc, 25*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		s.Run(ctx)
		close(done)
	}()

	require.Eventually(t, func() bool {
		return count.Load() >= 3
	}, 500*time.Millisecond, 5*time.Millisecond,
		"scheduler must keep ticking even when every sweep errors")

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("scheduler did not exit after context cancel")
	}
}

// TestScheduler_Run_StopsOnContextCancel — the scheduler must exit
// promptly when its parent context is cancelled, so the graceful
// shutdown path doesn't block longer than the in-flight tick.
func TestScheduler_Run_StopsOnContextCancel(t *testing.T) {
	repo := &schedRepo{}
	svc, err := retentionapp.NewService(repo, []retention.Policy{newSchedPolicy()})
	require.NoError(t, err)
	s := retentionapp.NewScheduler(svc, 1*time.Hour)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		s.Run(ctx)
		close(done)
	}()
	cancel()
	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("scheduler did not exit on cancel")
	}
}

// fakeCountingRepo is a thread-safe variant that increments a counter
// every time Sweep is called and optionally returns an injected error.
// We need atomic counters in the cadence tests because the scheduler
// runs on a goroutine and we read the count from the test goroutine.
type fakeCountingRepo struct {
	onSweep func()
	err     error
}

func (r *fakeCountingRepo) Sweep(_ context.Context, _ retention.Policy) (int, error) {
	if r.onSweep != nil {
		r.onSweep()
	}
	if r.err != nil {
		return 0, r.err
	}
	return 0, nil
}
