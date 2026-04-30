package webhookidempotency_test

// Unit tests for the composite claimer that backs BUG-10.
//
// Coverage requirements (from the brief):
//   - first webhook → processed
//   - replay → skipped (Redis hit)
//   - replay with Redis down → still skipped (Postgres catches it)
//   - replay with both Redis AND Postgres down → reject
//   - concurrent: 10 goroutines processing the same event_id → exactly
//     1 processed, 9 skipped
//
// The integration test that proves end-to-end behaviour against real
// Postgres + Redis lives in handler/stripe_handler_idempotency_test.go.

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	redisadapter "marketplace-backend/internal/adapter/redis"
	"marketplace-backend/internal/app/webhookidempotency"
)

// fakeCache stubs the Redis fast-path. Each call category is
// programmable so a single test can reproduce the full matrix
// (hit / miss / cache error / mark-seen error).
type fakeCache struct {
	mu        sync.Mutex
	tryClaims int
	markSeens int

	// claimResult drives TryCacheClaim. Set claimErr to a
	// *redis.CacheError to simulate a Redis outage.
	claimResult bool
	claimErr    error

	// markSeenErr drives MarkSeen. Almost always nil — the cache
	// populate is best-effort, but tests can wire it to a non-nil
	// error to assert that the surrounding flow logs and continues.
	markSeenErr error

	// seenIDs records which IDs had MarkSeen called so a test can
	// assert the cache was correctly populated AFTER Postgres returned.
	seenIDs []string
}

func (f *fakeCache) TryCacheClaim(_ context.Context, _ string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.tryClaims++
	return f.claimResult, f.claimErr
}

func (f *fakeCache) MarkSeen(_ context.Context, eventID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.markSeens++
	f.seenIDs = append(f.seenIDs, eventID)
	return f.markSeenErr
}

// fakeDurable stubs the Postgres source of truth.
type fakeDurable struct {
	mu     sync.Mutex
	calls  int
	seen   map[string]struct{} // event_ids already inserted
	err    error               // when non-nil, every call returns it
}

func newFakeDurable() *fakeDurable {
	return &fakeDurable{seen: map[string]struct{}{}}
}

func (f *fakeDurable) TryClaim(_ context.Context, eventID, _ string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if f.err != nil {
		return false, f.err
	}
	if _, exists := f.seen[eventID]; exists {
		return false, nil
	}
	f.seen[eventID] = struct{}{}
	return true, nil
}

// ---------------------------------------------------------------------------
// Constructor + input validation.
// ---------------------------------------------------------------------------

func TestNewClaimer_RequiresDurable(t *testing.T) {
	_, err := webhookidempotency.NewClaimer(nil, &fakeCache{})
	require.Error(t, err)
	assert.ErrorContains(t, err, "durable store is required")
}

func TestClaimer_RejectsEmptyEventID(t *testing.T) {
	c, err := webhookidempotency.NewClaimer(newFakeDurable(), &fakeCache{})
	require.NoError(t, err)

	claimed, err := c.TryClaim(context.Background(), "", "evt.type")
	require.Error(t, err)
	assert.False(t, claimed)
}

// ---------------------------------------------------------------------------
// Happy paths: first delivery, replay-via-cache, replay-via-postgres.
// ---------------------------------------------------------------------------

func TestClaimer_FirstDelivery_Processes(t *testing.T) {
	cache := &fakeCache{claimResult: true}
	durable := newFakeDurable()
	c, err := webhookidempotency.NewClaimer(durable, cache)
	require.NoError(t, err)

	claimed, err := c.TryClaim(context.Background(), "evt_first", "subscription.created")
	require.NoError(t, err)
	assert.True(t, claimed, "first delivery must claim")
	assert.Equal(t, 1, cache.tryClaims, "cache must be consulted")
	assert.Equal(t, 1, durable.calls, "durable must back-stop the cache claim")
	assert.Equal(t, 1, cache.markSeens, "cache must be populated after durable verdict")
	assert.Equal(t, []string{"evt_first"}, cache.seenIDs)
}

func TestClaimer_ReplayCacheHit_SkipsDurable(t *testing.T) {
	// claimResult=false simulates a Redis SETNX returning false → key
	// exists → we have processed this event in the last 5 minutes.
	cache := &fakeCache{claimResult: false}
	durable := newFakeDurable()
	c, err := webhookidempotency.NewClaimer(durable, cache)
	require.NoError(t, err)

	claimed, err := c.TryClaim(context.Background(), "evt_replay", "any")
	require.NoError(t, err)
	assert.False(t, claimed, "cache hit must short-circuit as duplicate")
	assert.Equal(t, 0, durable.calls, "durable MUST be skipped on cache hit (hot path)")
	assert.Equal(t, 0, cache.markSeens, "cache hit needs no second populate")
}

func TestClaimer_CacheMiss_DurableConfirmsReplay(t *testing.T) {
	// Cache miss (because TTL expired) but Postgres has seen this
	// event id before → still a duplicate.
	cache := &fakeCache{claimResult: true} // SETNX succeeded
	durable := newFakeDurable()
	durable.seen["evt_x"] = struct{}{} // already in DB

	c, err := webhookidempotency.NewClaimer(durable, cache)
	require.NoError(t, err)

	claimed, err := c.TryClaim(context.Background(), "evt_x", "any")
	require.NoError(t, err)
	assert.False(t, claimed, "durable replay verdict must override cache miss")
	assert.Equal(t, 1, durable.calls)
	assert.Equal(t, 1, cache.markSeens, "cache must be repopulated for next replay")
}

// ---------------------------------------------------------------------------
// Failure paths: Redis down, Postgres down, both down.
// ---------------------------------------------------------------------------

func TestClaimer_RedisDown_FallsThroughToDurable(t *testing.T) {
	// Redis returns *CacheError → composite must still consult durable.
	cache := &fakeCache{claimErr: &redisadapter.CacheError{Err: errors.New("connection refused")}}
	durable := newFakeDurable()
	c, err := webhookidempotency.NewClaimer(durable, cache)
	require.NoError(t, err)

	claimed, err := c.TryClaim(context.Background(), "evt_redis_down", "any")
	require.NoError(t, err, "Redis down must NOT fail the request — durable saves the day")
	assert.True(t, claimed, "first delivery resolved by durable layer")
	assert.Equal(t, 1, durable.calls)
}

func TestClaimer_RedisDown_DurableSeesReplay_StillSkips(t *testing.T) {
	// Critical BUG-10 invariant: Redis down + already-seen event MUST
	// still be detected as a replay. The pre-fix code returned
	// (true, err) on Redis errors, so the handler would re-process.
	cache := &fakeCache{claimErr: &redisadapter.CacheError{Err: errors.New("redis down")}}
	durable := newFakeDurable()
	durable.seen["evt_replay_during_outage"] = struct{}{}

	c, err := webhookidempotency.NewClaimer(durable, cache)
	require.NoError(t, err)

	claimed, err := c.TryClaim(context.Background(), "evt_replay_during_outage", "subscription.created")
	require.NoError(t, err)
	assert.False(t, claimed, "Postgres MUST catch the replay even with Redis offline — this is the BUG-10 fix")
}

func TestClaimer_BothLayersDown_ReturnsError(t *testing.T) {
	cache := &fakeCache{claimErr: &redisadapter.CacheError{Err: errors.New("redis down")}}
	durable := newFakeDurable()
	durable.err = errors.New("postgres down")

	c, err := webhookidempotency.NewClaimer(durable, cache)
	require.NoError(t, err)

	claimed, err := c.TryClaim(context.Background(), "evt_meltdown", "any")
	require.Error(t, err, "both layers down MUST surface as an error so the handler returns 503")
	assert.False(t, claimed, "no claim verdict when we cannot determine first/replay")
	assert.ErrorContains(t, err, "durable claim failed")
}

func TestClaimer_DurableOnlyFails_NoCache_StillReturnsError(t *testing.T) {
	// Some test setups don't wire a cache. Durable failure must still
	// propagate so the caller can react.
	durable := newFakeDurable()
	durable.err = errors.New("postgres down")

	c, err := webhookidempotency.NewClaimer(durable, nil)
	require.NoError(t, err, "nil cache is allowed; nil durable is not")

	claimed, err := c.TryClaim(context.Background(), "evt_x", "any")
	require.Error(t, err)
	assert.False(t, claimed)
}

func TestClaimer_NoCache_DurableHandlesEverything(t *testing.T) {
	// Exercise the cache-disabled path: every call goes straight to
	// Postgres. The two calls below first claim then replay.
	c, err := webhookidempotency.NewClaimer(newFakeDurable(), nil)
	require.NoError(t, err)

	first, err := c.TryClaim(context.Background(), "evt_solo", "any")
	require.NoError(t, err)
	assert.True(t, first)

	replay, err := c.TryClaim(context.Background(), "evt_solo", "any")
	require.NoError(t, err)
	assert.False(t, replay)
}

// ---------------------------------------------------------------------------
// Concurrency: 10 goroutines on the same event_id → exactly 1 processed.
// ---------------------------------------------------------------------------

func TestClaimer_Concurrent_SameEventID_OnlyOneClaims(t *testing.T) {
	const goroutines = 10
	cache := &fakeCache{claimResult: true} // pretend SETNX always succeeded
	durable := newFakeDurable()

	c, err := webhookidempotency.NewClaimer(durable, cache)
	require.NoError(t, err)

	var (
		processed atomic.Int32
		skipped   atomic.Int32
		errs      atomic.Int32
	)
	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			claimed, err := c.TryClaim(context.Background(), "evt_concurrent", "subscription.created")
			if err != nil {
				errs.Add(1)
				return
			}
			if claimed {
				processed.Add(1)
			} else {
				skipped.Add(1)
			}
		}()
	}
	wg.Wait()

	assert.Equal(t, int32(0), errs.Load(), "no errors on the happy concurrent path")
	assert.Equal(t, int32(1), processed.Load(),
		"exactly one goroutine must claim the event — UNIQUE in stripe_webhook_events enforces this")
	assert.Equal(t, int32(goroutines-1), skipped.Load(),
		"every other goroutine must observe the replay verdict")
}

// ---------------------------------------------------------------------------
// MarkSeen errors are non-fatal.
// ---------------------------------------------------------------------------

func TestClaimer_MarkSeenError_DoesNotFailRequest(t *testing.T) {
	cache := &fakeCache{
		claimResult: true,
		markSeenErr: errors.New("redis exploded after durable claim"),
	}
	durable := newFakeDurable()

	c, err := webhookidempotency.NewClaimer(durable, cache)
	require.NoError(t, err)

	claimed, err := c.TryClaim(context.Background(), "evt_mark_fails", "any")
	require.NoError(t, err, "cache populate failures must NEVER bubble up")
	assert.True(t, claimed)
}

// BUG-10: a non-CacheError from the cache (an unexpected error class
// the future adapter might emit) MUST trigger the default fallthrough
// branch, NOT short-circuit. The test passes a plain error so the
// switch enters the default arm and falls through to durable.
func TestClaimer_UnexpectedCacheErrorClass_FallsThroughToDurable(t *testing.T) {
	// Plain (non-*CacheError) error — covers the `default:` branch.
	cache := &fakeCache{claimErr: errors.New("a future error class we did not anticipate")}
	durable := newFakeDurable()

	c, err := webhookidempotency.NewClaimer(durable, cache)
	require.NoError(t, err)

	claimed, err := c.TryClaim(context.Background(), "evt_unknown_err", "any")
	require.NoError(t, err, "unknown cache error class must defer to durable, not fail loud")
	assert.True(t, claimed, "durable layer succeeds → first delivery wins")
	assert.Equal(t, 1, durable.calls, "durable must be consulted on unknown cache error")
}

// Empty-string event id error message is part of the contract — pin it.
func TestClaimer_EmptyEventID_ErrorMessage(t *testing.T) {
	c, err := webhookidempotency.NewClaimer(newFakeDurable(), &fakeCache{})
	require.NoError(t, err)

	_, err = c.TryClaim(context.Background(), "", "any")
	require.Error(t, err)
	assert.ErrorContains(t, err, "empty event_id")
}
