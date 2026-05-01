package postgres_test

// Integration tests for BUG-NEW-03 — pending_events stuck forever in
// 'processing' after a worker crash. Gated behind MARKETPLACE_TEST_DATABASE_URL
// like every other postgres integration test.
//
// The fix:
//   1. PopDue claims rows in 'processing' whose updated_at is older
//      than a configurable stale threshold (default 5 minutes).
//   2. PopDue refreshes updated_at on every claim so concurrent workers
//      can't re-claim a row that's actively being worked on.
//
// Run:
//
//	MARKETPLACE_TEST_DATABASE_URL=postgres://postgres:postgres@localhost:5435/marketplace_go_bugs_high?sslmode=disable \
//	  go test ./internal/adapter/postgres/ -run TestPendingEventRepository_StaleRecovery -count=1 -race

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/adapter/postgres"
	"marketplace-backend/internal/domain/pendingevent"
)

// TestPendingEventRepository_StaleRecovery_ClaimsStaleProcessingRows pins
// the BUG-NEW-03 fix: a worker crash leaves a row in 'processing' with
// no recovery path. After the fix, the next PopDue call (using a stale
// threshold > the crash window) reclaims it.
//
// The scenario:
//   1. INSERT a row directly in 'processing' status with updated_at in
//      the past (simulating a worker that claimed the row and crashed).
//   2. PopDue with a 100ms stale threshold (deterministic for the test).
//   3. Assert the row IS reclaimed and processed.
func TestPendingEventRepository_StaleRecovery_ClaimsStaleProcessingRows(t *testing.T) {
	db := testDB(t)
	cleanupPendingEvents(t)
	repo := postgres.NewPendingEventRepository(db)
	ctx := context.Background()

	// Insert a row directly in 'processing' status with updated_at in
	// the past — simulating a worker that crashed mid-handler. attempts=1
	// since it was already claimed once.
	staleID := uuid.New()
	stalePayload := []byte(`{"test_id":"stale"}`)
	now := time.Now()
	staleUpdatedAt := now.Add(-2 * time.Second)
	_, err := db.ExecContext(ctx, `
		INSERT INTO pending_events
		    (id, event_type, payload, fires_at, status, attempts, last_error, processed_at, created_at, updated_at)
		VALUES
		    ($1, $2, $3, $4, 'processing', 1, NULL, NULL, $5, $6)
	`, staleID, string(pendingevent.TypeMilestoneAutoApprove), stalePayload, now.Add(-1*time.Hour), now.Add(-1*time.Hour), staleUpdatedAt)
	require.NoError(t, err)

	// PopDue with a 1s stale threshold — the row's updated_at is 2s ago
	// so it MUST be reclaimed.
	popped, err := repo.PopDueWithStaleThreshold(ctx, 10, 1*time.Second)
	require.NoError(t, err)
	require.Len(t, popped, 1, "BUG-NEW-03: stale processing row must be reclaimed")
	assert.Equal(t, staleID, popped[0].ID)
	assert.Equal(t, pendingevent.StatusProcessing, popped[0].Status, "still processing after re-claim")
	assert.Equal(t, 2, popped[0].Attempts, "attempts bumped to 2 (the second claim)")
}

// TestPendingEventRepository_StaleRecovery_FreshProcessingRowsAreSafe is
// the regression: a row in 'processing' whose updated_at is recent
// (handler still running) MUST NOT be re-claimed. Otherwise concurrent
// workers would fight over the same in-flight row.
func TestPendingEventRepository_StaleRecovery_FreshProcessingRowsAreSafe(t *testing.T) {
	db := testDB(t)
	cleanupPendingEvents(t)
	repo := postgres.NewPendingEventRepository(db)
	ctx := context.Background()

	// Insert a row in 'processing' with updated_at = now (handler just
	// claimed it and is mid-flight).
	freshID := uuid.New()
	freshPayload := []byte(`{"test_id":"fresh"}`)
	now := time.Now()
	_, err := db.ExecContext(ctx, `
		INSERT INTO pending_events
		    (id, event_type, payload, fires_at, status, attempts, last_error, processed_at, created_at, updated_at)
		VALUES
		    ($1, $2, $3, $4, 'processing', 1, NULL, NULL, $5, now())
	`, freshID, string(pendingevent.TypeStripeTransfer), freshPayload, now.Add(-1*time.Hour), now.Add(-1*time.Hour))
	require.NoError(t, err)

	// PopDue with a 5s stale threshold — the row's updated_at is well
	// under that, so it MUST NOT be reclaimed.
	popped, err := repo.PopDueWithStaleThreshold(ctx, 10, 5*time.Second)
	require.NoError(t, err)
	assert.Empty(t, popped, "fresh processing row must NOT be reclaimed (worker still owns it)")
}

// TestPendingEventRepository_StaleRecovery_ReclaimRefreshesUpdatedAt
// asserts the second-claim race protection: when a stale row is
// reclaimed, its updated_at is refreshed to NOW. So a SECOND concurrent
// worker arriving immediately after the first MUST see the row as
// fresh and skip it — they don't both grab the same stale row.
func TestPendingEventRepository_StaleRecovery_ReclaimRefreshesUpdatedAt(t *testing.T) {
	db := testDB(t)
	cleanupPendingEvents(t)
	repo := postgres.NewPendingEventRepository(db)
	ctx := context.Background()

	staleID := uuid.New()
	stalePayload := []byte(`{"test_id":"reclaim"}`)
	now := time.Now()
	staleUpdatedAt := now.Add(-10 * time.Second)
	_, err := db.ExecContext(ctx, `
		INSERT INTO pending_events
		    (id, event_type, payload, fires_at, status, attempts, last_error, processed_at, created_at, updated_at)
		VALUES
		    ($1, $2, $3, $4, 'processing', 1, NULL, NULL, $5, $6)
	`, staleID, string(pendingevent.TypeMilestoneAutoApprove), stalePayload, now.Add(-1*time.Hour), now.Add(-1*time.Hour), staleUpdatedAt)
	require.NoError(t, err)

	// First worker reclaims the stale row.
	popped, err := repo.PopDueWithStaleThreshold(ctx, 10, 1*time.Second)
	require.NoError(t, err)
	require.Len(t, popped, 1, "first worker reclaims the stale row")
	assert.Equal(t, staleID, popped[0].ID)

	// Second worker arriving immediately MUST NOT re-claim it — the
	// PopDue UPDATE refreshed updated_at to NOW, so the row is fresh.
	poppedAgain, err := repo.PopDueWithStaleThreshold(ctx, 10, 1*time.Second)
	require.NoError(t, err)
	assert.Empty(t, poppedAgain, "BUG-NEW-03: re-claim must refresh updated_at so a concurrent worker does not double-pop")
}

// TestPendingEventRepository_StaleRecovery_RespectsMaxAttempts asserts
// the recovery path respects the same MaxAttempts cap as the normal
// pending|failed path. A row stuck in 'processing' with attempts >= 5
// must NOT be reclaimed — it has exhausted retries.
func TestPendingEventRepository_StaleRecovery_RespectsMaxAttempts(t *testing.T) {
	db := testDB(t)
	cleanupPendingEvents(t)
	repo := postgres.NewPendingEventRepository(db)
	ctx := context.Background()

	exhaustedID := uuid.New()
	payload := []byte(`{"test_id":"exhausted"}`)
	now := time.Now()
	staleUpdatedAt := now.Add(-10 * time.Second)
	_, err := db.ExecContext(ctx, `
		INSERT INTO pending_events
		    (id, event_type, payload, fires_at, status, attempts, last_error, processed_at, created_at, updated_at)
		VALUES
		    ($1, $2, $3, $4, 'processing', 5, NULL, NULL, $5, $6)
	`, exhaustedID, string(pendingevent.TypeMilestoneAutoApprove), payload, now.Add(-1*time.Hour), now.Add(-1*time.Hour), staleUpdatedAt)
	require.NoError(t, err)

	popped, err := repo.PopDueWithStaleThreshold(ctx, 10, 1*time.Second)
	require.NoError(t, err)
	assert.Empty(t, popped, "exhausted-retry row in 'processing' must NOT be reclaimed (would loop forever)")
}
