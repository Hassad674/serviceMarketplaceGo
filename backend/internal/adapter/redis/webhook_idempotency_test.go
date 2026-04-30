package redis_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	adapter "marketplace-backend/internal/adapter/redis"
)

func newIdempotencyTest(t *testing.T, ttl time.Duration) (*adapter.WebhookIdempotencyStore, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return adapter.NewWebhookIdempotencyStore(client, ttl), mr
}

func TestIdempotency_FirstClaimWinsSecondIsNoop(t *testing.T) {
	store, _ := newIdempotencyTest(t, time.Minute)

	first, err := store.TryCacheClaim(context.Background(), "evt_123")
	require.NoError(t, err)
	assert.True(t, first, "first call must win the claim")

	second, err := store.TryCacheClaim(context.Background(), "evt_123")
	require.NoError(t, err)
	assert.False(t, second, "replay of the same event_id must not claim")
}

func TestIdempotency_DifferentEventsDoNotCollide(t *testing.T) {
	store, _ := newIdempotencyTest(t, time.Minute)

	a, _ := store.TryCacheClaim(context.Background(), "evt_a")
	b, _ := store.TryCacheClaim(context.Background(), "evt_b")

	assert.True(t, a)
	assert.True(t, b)
}

func TestIdempotency_ClaimExpiresAfterTTL(t *testing.T) {
	store, mr := newIdempotencyTest(t, time.Minute)

	first, _ := store.TryCacheClaim(context.Background(), "evt_exp")
	assert.True(t, first)

	mr.FastForward(2 * time.Minute)

	again, _ := store.TryCacheClaim(context.Background(), "evt_exp")
	assert.True(t, again, "after TTL the event_id is claimable again")
}

func TestIdempotency_EmptyEventID_AlwaysClaims(t *testing.T) {
	// Defensive: a missing event id MUST never be stored (would
	// collide with every other empty-id request) and the caller sees
	// "claimed" so processing proceeds.
	store, _ := newIdempotencyTest(t, time.Minute)

	a, _ := store.TryCacheClaim(context.Background(), "")
	b, _ := store.TryCacheClaim(context.Background(), "")

	assert.True(t, a)
	assert.True(t, b)
}

func TestIdempotency_ZeroTTLDefaults(t *testing.T) {
	store, mr := newIdempotencyTest(t, 0)

	_, err := store.TryCacheClaim(context.Background(), "evt_ttl_default")
	require.NoError(t, err)
	ttl := mr.TTL("stripe:event:evt_ttl_default")
	assert.Greater(t, ttl, time.Minute, "zero TTL MUST fall back to a sensible default")
}

// BUG-10: When Redis is unavailable, TryCacheClaim must surface the
// error as *CacheError so the composite claimer falls through to the
// durable Postgres path. The pre-fix behaviour returned (true, err)
// which caused double-processing on cache outages.
func TestIdempotency_CacheError_SurfacesAsCacheError(t *testing.T) {
	store, mr := newIdempotencyTest(t, time.Minute)
	// Tear down Redis so the next call fails.
	mr.Close()

	claimed, err := store.TryCacheClaim(context.Background(), "evt_redis_down")
	require.Error(t, err)
	assert.False(t, claimed, "cache error must NOT report a claim — caller must consult Postgres")
	var cacheErr *adapter.CacheError
	assert.ErrorAs(t, err, &cacheErr, "wrapped error must be *redis.CacheError so callers can detect cache faults")
}

func TestIdempotency_MarkSeen_PopulatesCache(t *testing.T) {
	store, mr := newIdempotencyTest(t, time.Minute)
	require.NoError(t, store.MarkSeen(context.Background(), "evt_marked"))

	// Subsequent TryCacheClaim must report the event has been seen.
	claimed, err := store.TryCacheClaim(context.Background(), "evt_marked")
	require.NoError(t, err)
	assert.False(t, claimed, "MarkSeen must seed the cache so replays short-circuit")
	assert.True(t, mr.Exists("stripe:event:evt_marked"))
}

func TestIdempotency_MarkSeen_EmptyIDIsNoop(t *testing.T) {
	store, _ := newIdempotencyTest(t, time.Minute)
	require.NoError(t, store.MarkSeen(context.Background(), ""))
}

func TestIdempotency_MarkSeen_RedisDown_ReturnsCacheError(t *testing.T) {
	store, mr := newIdempotencyTest(t, time.Minute)
	mr.Close()

	err := store.MarkSeen(context.Background(), "evt_seen")
	require.Error(t, err)
	var cacheErr *adapter.CacheError
	assert.ErrorAs(t, err, &cacheErr)
}
