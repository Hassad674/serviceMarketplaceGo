package postgres_test

// Integration tests for the postgres.TxRunner — gated behind
// MARKETPLACE_TEST_DATABASE_URL like the rest of the package, so the
// suite skips on a fresh checkout that has no running Postgres.
//
// Coverage focuses on the contract that BUG-05 leans on:
//   - fn nil → returns a clear error (programmer mistake)
//   - fn returns nil → the transaction commits, writes are visible
//   - fn returns non-nil → the transaction rolls back, writes vanish
//   - simultaneous TxRunner.RunInTx calls do not share a transaction

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/adapter/postgres"
	"marketplace-backend/internal/domain/pendingevent"
)

// scheduleMarker inserts a pending_events row inside the given tx so
// the surrounding test can use it as a commit-or-rollback marker.
func scheduleMarker(t *testing.T, ctx context.Context, tx *sql.Tx, db *sql.DB) uuid.UUID {
	t.Helper()
	repo := postgres.NewPendingEventRepository(db)
	e := newTestPendingEvent(t, pendingevent.TypeStripeTransfer, time.Now().Add(1*time.Hour))
	require.NoError(t, repo.ScheduleTx(ctx, tx, e))
	return e.ID
}

func TestTxRunner_NilFnReturnsError(t *testing.T) {
	db := testDB(t)
	runner := postgres.NewTxRunner(db)

	err := runner.RunInTx(context.Background(), nil)
	assert.ErrorContains(t, err, "fn is required")
}

func TestTxRunner_HappyPath_Commits(t *testing.T) {
	db := testDB(t)
	cleanupPendingEvents(t)
	runner := postgres.NewTxRunner(db)
	ctx := context.Background()

	var insertedID uuid.UUID
	err := runner.RunInTx(ctx, func(tx *sql.Tx) error {
		insertedID = scheduleMarker(t, ctx, tx, db)
		return nil
	})
	require.NoError(t, err)

	// The row must be visible on a fresh connection — proof of commit.
	repo := postgres.NewPendingEventRepository(db)
	got, err := repo.GetByID(ctx, insertedID)
	require.NoError(t, err)
	assert.Equal(t, insertedID, got.ID)
}

func TestTxRunner_FnError_RollsBack(t *testing.T) {
	db := testDB(t)
	cleanupPendingEvents(t)
	runner := postgres.NewTxRunner(db)
	ctx := context.Background()

	var insertedID uuid.UUID
	rollbackErr := errors.New("simulated handler failure")
	err := runner.RunInTx(ctx, func(tx *sql.Tx) error {
		insertedID = scheduleMarker(t, ctx, tx, db)
		// returning a non-nil error must cause Rollback — the row
		// from scheduleMarker must NOT be visible afterwards.
		return rollbackErr
	})
	require.ErrorIs(t, err, rollbackErr)

	repo := postgres.NewPendingEventRepository(db)
	_, err = repo.GetByID(ctx, insertedID)
	assert.ErrorIs(t, err, pendingevent.ErrEventNotFound,
		"rolled-back row must not survive in pending_events")
}

func TestTxRunner_ConcurrentCallsDontShareTx(t *testing.T) {
	db := testDB(t)
	cleanupPendingEvents(t)
	runner := postgres.NewTxRunner(db)
	ctx := context.Background()

	const callers = 8
	var (
		wg  sync.WaitGroup
		mu  sync.Mutex
		ids []uuid.UUID
	)
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := runner.RunInTx(ctx, func(tx *sql.Tx) error {
				id := scheduleMarker(t, ctx, tx, db)
				mu.Lock()
				ids = append(ids, id)
				mu.Unlock()
				return nil
			})
			assert.NoError(t, err)
		}()
	}
	wg.Wait()

	// Every concurrent caller must have inserted independently.
	repo := postgres.NewPendingEventRepository(db)
	require.Len(t, ids, callers)
	for _, id := range ids {
		_, err := repo.GetByID(ctx, id)
		require.NoError(t, err, "row %s must have committed", id)
	}
}
