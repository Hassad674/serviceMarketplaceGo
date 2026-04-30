package postgres_test

// Integration tests for PendingEventRepository (migration 087 schema).
// Gated behind MARKETPLACE_TEST_DATABASE_URL — auto-skip when unset.
//
// Run against the isolated milestones DB copy:
//
//	MARKETPLACE_TEST_DATABASE_URL=postgres://postgres:postgres@localhost:5435/marketplace_go_milestones?sslmode=disable \
//	  go test ./internal/adapter/postgres/ -run TestPendingEventRepository -count=1

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/adapter/postgres"
	"marketplace-backend/internal/domain/pendingevent"
)

// newTestPendingEvent is a local test helper that builds a domain
// pendingevent ready for INSERT, with fires_at slightly in the past
// so PopDue picks it up immediately.
func newTestPendingEvent(t *testing.T, eventType pendingevent.EventType, firesAt time.Time) *pendingevent.PendingEvent {
	t.Helper()
	payload, err := json.Marshal(map[string]string{"test_id": uuid.NewString()})
	require.NoError(t, err)
	e, err := pendingevent.NewPendingEvent(pendingevent.NewPendingEventInput{
		EventType: eventType,
		Payload:   payload,
		FiresAt:   firesAt,
	})
	require.NoError(t, err)
	return e
}

// cleanupPendingEvents removes any test rows the suite may have
// created in a previous run so the assertions stay deterministic.
// Includes search.* event types so the BUG-05 outbox integration
// tests don't leak rows into the broader pending_events suite.
func cleanupPendingEvents(t *testing.T) {
	t.Helper()
	db := testDB(t)
	_, _ = db.Exec(`DELETE FROM pending_events WHERE event_type IN ('milestone_auto_approve', 'milestone_fund_reminder', 'proposal_auto_close', 'stripe_transfer', 'search.reindex', 'search.delete')`)
}

func TestPendingEventRepository_ScheduleAndGetByID(t *testing.T) {
	db := testDB(t)
	cleanupPendingEvents(t)
	repo := postgres.NewPendingEventRepository(db)
	ctx := context.Background()

	e := newTestPendingEvent(t, pendingevent.TypeMilestoneAutoApprove, time.Now().Add(1*time.Hour))
	require.NoError(t, repo.Schedule(ctx, e))

	got, err := repo.GetByID(ctx, e.ID)
	require.NoError(t, err)
	assert.Equal(t, e.ID, got.ID)
	assert.Equal(t, pendingevent.TypeMilestoneAutoApprove, got.EventType)
	assert.Equal(t, pendingevent.StatusPending, got.Status)
	assert.Equal(t, 0, got.Attempts)
	assert.Nil(t, got.LastError)
}

func TestPendingEventRepository_GetByID_NotFound(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewPendingEventRepository(db)
	_, err := repo.GetByID(context.Background(), uuid.New())
	require.ErrorIs(t, err, pendingevent.ErrEventNotFound)
}

func TestPendingEventRepository_PopDue_OnlyClaimsDueRows(t *testing.T) {
	db := testDB(t)
	cleanupPendingEvents(t)
	repo := postgres.NewPendingEventRepository(db)
	ctx := context.Background()

	// One event due in the past, one in the future.
	pastEvent := newTestPendingEvent(t, pendingevent.TypeMilestoneAutoApprove, time.Now().Add(-1*time.Minute))
	futureEvent := newTestPendingEvent(t, pendingevent.TypeMilestoneAutoApprove, time.Now().Add(1*time.Hour))
	require.NoError(t, repo.Schedule(ctx, pastEvent))
	require.NoError(t, repo.Schedule(ctx, futureEvent))

	popped, err := repo.PopDue(ctx, 10)
	require.NoError(t, err)
	require.Len(t, popped, 1, "only the past-due event should be claimed")
	assert.Equal(t, pastEvent.ID, popped[0].ID)
	assert.Equal(t, pendingevent.StatusProcessing, popped[0].Status, "claimed events must be in processing status")
	assert.Equal(t, 1, popped[0].Attempts, "attempts must be bumped to 1 inside the pop transaction")

	// Future event should still be pending and untouched.
	stillFuture, err := repo.GetByID(ctx, futureEvent.ID)
	require.NoError(t, err)
	assert.Equal(t, pendingevent.StatusPending, stillFuture.Status)
}

func TestPendingEventRepository_PopDue_MarkDone(t *testing.T) {
	db := testDB(t)
	cleanupPendingEvents(t)
	repo := postgres.NewPendingEventRepository(db)
	ctx := context.Background()

	e := newTestPendingEvent(t, pendingevent.TypeStripeTransfer, time.Now().Add(-1*time.Minute))
	require.NoError(t, repo.Schedule(ctx, e))

	popped, err := repo.PopDue(ctx, 10)
	require.NoError(t, err)
	require.Len(t, popped, 1)

	require.NoError(t, repo.MarkDone(ctx, popped[0]))

	final, err := repo.GetByID(ctx, e.ID)
	require.NoError(t, err)
	assert.Equal(t, pendingevent.StatusDone, final.Status)
	assert.NotNil(t, final.ProcessedAt)
	assert.Nil(t, final.LastError)
}

func TestPendingEventRepository_PopDue_MarkFailed_ReschedulesViaBackoff(t *testing.T) {
	db := testDB(t)
	cleanupPendingEvents(t)
	repo := postgres.NewPendingEventRepository(db)
	ctx := context.Background()

	e := newTestPendingEvent(t, pendingevent.TypeMilestoneFundReminder, time.Now().Add(-1*time.Minute))
	require.NoError(t, repo.Schedule(ctx, e))

	popped, err := repo.PopDue(ctx, 10)
	require.NoError(t, err)
	require.Len(t, popped, 1)
	popped0 := popped[0]

	// Apply the domain MarkFailed transition (sets backoff fires_at).
	require.NoError(t, popped0.MarkFailed(errors.New("simulated handler failure")))
	require.NoError(t, repo.MarkFailed(ctx, popped0))

	final, err := repo.GetByID(ctx, e.ID)
	require.NoError(t, err)
	assert.Equal(t, pendingevent.StatusFailed, final.Status)
	assert.NotNil(t, final.LastError)
	assert.Equal(t, "simulated handler failure", *final.LastError)
	// fires_at should now be ~1 minute in the future (first backoff).
	assert.WithinDuration(t, time.Now().Add(1*time.Minute), final.FiresAt, 5*time.Second)
}

func TestPendingEventRepository_PopDue_FailedRowsAreRetried(t *testing.T) {
	db := testDB(t)
	cleanupPendingEvents(t)
	repo := postgres.NewPendingEventRepository(db)
	ctx := context.Background()

	// Create + immediately set to failed with fires_at in the past
	// so PopDue picks it up again.
	e := newTestPendingEvent(t, pendingevent.TypeMilestoneAutoApprove, time.Now().Add(-1*time.Hour))
	require.NoError(t, repo.Schedule(ctx, e))

	popped, err := repo.PopDue(ctx, 10)
	require.NoError(t, err)
	require.Len(t, popped, 1)

	// Fail it once.
	require.NoError(t, popped[0].MarkFailed(errors.New("transient")))
	// Force fires_at into the past so the retry is immediately due.
	popped[0].FiresAt = time.Now().Add(-1 * time.Second)
	require.NoError(t, repo.MarkFailed(ctx, popped[0]))

	// Retry: PopDue should pick the same row up again.
	retry, err := repo.PopDue(ctx, 10)
	require.NoError(t, err)
	require.Len(t, retry, 1)
	assert.Equal(t, e.ID, retry[0].ID)
	assert.Equal(t, 2, retry[0].Attempts, "attempts should be bumped to 2 on the second pop")
}

func TestPendingEventRepository_PopDue_ConcurrentWorkersDontDoublePop(t *testing.T) {
	db := testDB(t)
	cleanupPendingEvents(t)
	repo := postgres.NewPendingEventRepository(db)
	ctx := context.Background()

	// Schedule 5 due events; spawn 4 workers each calling PopDue
	// at the same moment. The union of their results must contain
	// each event exactly once — no row may be claimed twice.
	const eventCount = 5
	const workerCount = 4
	var ids []uuid.UUID
	for i := 0; i < eventCount; i++ {
		e := newTestPendingEvent(t, pendingevent.TypeStripeTransfer, time.Now().Add(-1*time.Minute))
		require.NoError(t, repo.Schedule(ctx, e))
		ids = append(ids, e.ID)
	}

	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		claimed = map[uuid.UUID]int{}
	)
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			popped, err := repo.PopDue(ctx, 10)
			if err != nil {
				t.Errorf("worker pop: %v", err)
				return
			}
			mu.Lock()
			defer mu.Unlock()
			for _, e := range popped {
				claimed[e.ID]++
			}
		}()
	}
	wg.Wait()

	// Every event must have been claimed exactly once.
	require.Len(t, claimed, eventCount, "every scheduled event should have been claimed")
	for id, count := range claimed {
		assert.Equal(t, 1, count, "event %s claimed %d times, want 1", id, count)
	}
}
