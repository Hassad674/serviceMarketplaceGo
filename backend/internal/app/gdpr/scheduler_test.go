package gdpr

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScheduler_RefusesToRunWithoutSalt(t *testing.T) {
	repo := &stubGDPRRepo{}
	svc := newServiceForTest(t, ServiceDeps{Repo: repo, Users: &stubUserRepo{}})

	sch := NewScheduler(svc, "", 10, 10*time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	sch.Run(ctx)
	// Should return immediately when salt is empty — no need to
	// actually wait for the deadline because the function early-returns.
}

func TestScheduler_TicksImmediatelyOnStart(t *testing.T) {
	calls := atomic.Int64{}
	repo := &stubGDPRRepo{
		listPurgeableFn: func(_ context.Context, _ time.Time, _ int) ([]uuid.UUID, error) {
			calls.Add(1)
			return nil, nil
		},
	}
	svc := newServiceForTest(t, ServiceDeps{Repo: repo, Users: &stubUserRepo{}})

	sch := NewScheduler(svc, "salt", 10, 1*time.Hour) // long interval

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	sch.Run(ctx)

	assert.GreaterOrEqual(t, calls.Load(), int64(1), "scheduler must tick immediately on start")
}

func TestScheduler_TicksRepeatedly(t *testing.T) {
	calls := atomic.Int64{}
	repo := &stubGDPRRepo{
		listPurgeableFn: func(_ context.Context, _ time.Time, _ int) ([]uuid.UUID, error) {
			calls.Add(1)
			return nil, nil
		},
	}
	svc := newServiceForTest(t, ServiceDeps{Repo: repo, Users: &stubUserRepo{}})

	sch := NewScheduler(svc, "salt", 10, 10*time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Millisecond)
	defer cancel()
	sch.Run(ctx)

	// 60ms / 10ms = 6 ticks expected (with the immediate first tick).
	// Allow some slack for CI timing.
	assert.GreaterOrEqual(t, calls.Load(), int64(2),
		"scheduler should tick at least twice in 60ms with a 10ms interval")
}

func TestScheduler_PurgesEligibleRows(t *testing.T) {
	id := uuid.New()
	listed := atomic.Int64{}
	purged := atomic.Int64{}

	repo := &stubGDPRRepo{
		listPurgeableFn: func(_ context.Context, _ time.Time, _ int) ([]uuid.UUID, error) {
			n := listed.Add(1)
			if n == 1 {
				return []uuid.UUID{id}, nil
			}
			return nil, nil
		},
		purgeFn: func(_ context.Context, _ uuid.UUID, _ time.Time, salt string) (bool, error) {
			require.Equal(t, "salt-x", salt)
			purged.Add(1)
			return true, nil
		},
	}
	svc := newServiceForTest(t, ServiceDeps{Repo: repo, Users: &stubUserRepo{}})

	sch := NewScheduler(svc, "salt-x", 10, 10*time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	sch.Run(ctx)

	assert.Equal(t, int64(1), purged.Load(), "exactly one purge call expected")
}

func TestScheduler_StopsOnContextCancel(t *testing.T) {
	repo := &stubGDPRRepo{
		listPurgeableFn: func(_ context.Context, _ time.Time, _ int) ([]uuid.UUID, error) {
			return nil, nil
		},
	}
	svc := newServiceForTest(t, ServiceDeps{Repo: repo, Users: &stubUserRepo{}})

	sch := NewScheduler(svc, "salt", 10, 10*time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		sch.Run(ctx)
		close(done)
	}()

	cancel()
	select {
	case <-done:
		// good
	case <-time.After(200 * time.Millisecond):
		t.Fatal("scheduler did not stop after context cancel")
	}
}

func TestNewScheduler_DefaultInterval(t *testing.T) {
	sch := NewScheduler(nil, "salt", 0, 0)
	assert.Equal(t, SchedulerInterval, sch.interval, "zero interval falls back to default")
	assert.Equal(t, 100, sch.batchSize, "zero batch size falls back to 100")
}
