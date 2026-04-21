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

	first, err := store.TryClaim(context.Background(), "evt_123")
	require.NoError(t, err)
	assert.True(t, first, "first call must win the claim")

	second, err := store.TryClaim(context.Background(), "evt_123")
	require.NoError(t, err)
	assert.False(t, second, "replay of the same event_id must not claim")
}

func TestIdempotency_DifferentEventsDoNotCollide(t *testing.T) {
	store, _ := newIdempotencyTest(t, time.Minute)

	a, _ := store.TryClaim(context.Background(), "evt_a")
	b, _ := store.TryClaim(context.Background(), "evt_b")

	assert.True(t, a)
	assert.True(t, b)
}

func TestIdempotency_ClaimExpiresAfterTTL(t *testing.T) {
	store, mr := newIdempotencyTest(t, time.Minute)

	first, _ := store.TryClaim(context.Background(), "evt_exp")
	assert.True(t, first)

	mr.FastForward(2 * time.Minute)

	again, _ := store.TryClaim(context.Background(), "evt_exp")
	assert.True(t, again, "after TTL the event_id is claimable again")
}

func TestIdempotency_EmptyEventID_AlwaysClaims(t *testing.T) {
	// Defensive: a missing event id MUST never be stored (would
	// collide with every other empty-id request) and the caller sees
	// "claimed" so processing proceeds.
	store, _ := newIdempotencyTest(t, time.Minute)

	a, _ := store.TryClaim(context.Background(), "")
	b, _ := store.TryClaim(context.Background(), "")

	assert.True(t, a)
	assert.True(t, b)
}

func TestIdempotency_ZeroTTLDefaults(t *testing.T) {
	store, mr := newIdempotencyTest(t, 0)

	_, err := store.TryClaim(context.Background(), "evt_ttl_default")
	require.NoError(t, err)
	ttl := mr.TTL("stripe:event:evt_ttl_default")
	assert.Greater(t, ttl, time.Hour, "zero TTL MUST fall back to a multi-day default")
}
