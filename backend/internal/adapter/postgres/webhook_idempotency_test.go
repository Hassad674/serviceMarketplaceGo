package postgres_test

// Integration tests for the durable Postgres webhook-dedup adapter
// (BUG-10). Skips on a fresh checkout per the
// MARKETPLACE_TEST_DATABASE_URL convention.

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/adapter/postgres"
)

func cleanupWebhookEvents(t *testing.T, eventIDs ...string) {
	t.Helper()
	if len(eventIDs) == 0 {
		return
	}
	db := testDB(t)
	for _, id := range eventIDs {
		_, _ = db.Exec(`DELETE FROM stripe_webhook_events WHERE stripe_event_id = $1`, id)
	}
}

func TestPostgresIdempotency_FirstClaimWins(t *testing.T) {
	db := testDB(t)
	store := postgres.NewWebhookIdempotencyStore(db)
	const eventID = "evt_pg_first_claim"
	t.Cleanup(func() { cleanupWebhookEvents(t, eventID) })

	first, err := store.TryClaim(context.Background(), eventID, "subscription.created")
	require.NoError(t, err)
	assert.True(t, first, "first call must claim")
}

func TestPostgresIdempotency_ReplayDetected(t *testing.T) {
	db := testDB(t)
	store := postgres.NewWebhookIdempotencyStore(db)
	const eventID = "evt_pg_replay"
	t.Cleanup(func() { cleanupWebhookEvents(t, eventID) })

	first, err := store.TryClaim(context.Background(), eventID, "subscription.created")
	require.NoError(t, err)
	require.True(t, first)

	replay, err := store.TryClaim(context.Background(), eventID, "subscription.created")
	require.NoError(t, err, "ON CONFLICT DO NOTHING must NOT raise an error")
	assert.False(t, replay, "second claim of the same event_id must report replay")
}

func TestPostgresIdempotency_DifferentEventsDontCollide(t *testing.T) {
	db := testDB(t)
	store := postgres.NewWebhookIdempotencyStore(db)
	t.Cleanup(func() { cleanupWebhookEvents(t, "evt_pg_a", "evt_pg_b") })

	a, err := store.TryClaim(context.Background(), "evt_pg_a", "x")
	require.NoError(t, err)
	assert.True(t, a)

	b, err := store.TryClaim(context.Background(), "evt_pg_b", "y")
	require.NoError(t, err)
	assert.True(t, b)
}

func TestPostgresIdempotency_RejectsEmptyEventID(t *testing.T) {
	db := testDB(t)
	store := postgres.NewWebhookIdempotencyStore(db)

	_, err := store.TryClaim(context.Background(), "", "any")
	require.Error(t, err)
	assert.ErrorContains(t, err, "empty event_id")
}

// Concurrent claim — Postgres unique constraint must guarantee that
// exactly one out of N concurrent INSERTs sees the "first" verdict.
func TestPostgresIdempotency_Concurrent_OnlyOneClaims(t *testing.T) {
	db := testDB(t)
	store := postgres.NewWebhookIdempotencyStore(db)
	const eventID = "evt_pg_concurrent"
	const N = 16
	t.Cleanup(func() { cleanupWebhookEvents(t, eventID) })

	var (
		processed atomic.Int32
		skipped   atomic.Int32
		errs      atomic.Int32
	)
	var wg sync.WaitGroup
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			first, err := store.TryClaim(context.Background(), eventID, "any")
			if err != nil {
				errs.Add(1)
				return
			}
			if first {
				processed.Add(1)
			} else {
				skipped.Add(1)
			}
		}()
	}
	wg.Wait()

	assert.Equal(t, int32(0), errs.Load())
	assert.Equal(t, int32(1), processed.Load(),
		"unique constraint must select exactly one winner among concurrent INSERTs")
	assert.Equal(t, int32(N-1), skipped.Load())
}
