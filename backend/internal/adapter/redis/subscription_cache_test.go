package redis_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	adapter "marketplace-backend/internal/adapter/redis"
)

// stubReader is a controllable SubscriptionReader used to assert the
// cache's delegation behaviour and count inner calls.
type stubReader struct {
	active   bool
	err      error
	callsFor map[uuid.UUID]int
}

func newStub(active bool, err error) *stubReader {
	return &stubReader{
		active:   active,
		err:      err,
		callsFor: make(map[uuid.UUID]int),
	}
}

func (s *stubReader) IsActive(_ context.Context, userID uuid.UUID) (bool, error) {
	s.callsFor[userID]++
	return s.active, s.err
}

func newTestCache(t *testing.T, inner *stubReader, ttl time.Duration) (*adapter.CachedSubscriptionReader, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	return adapter.NewCachedSubscriptionReader(client, inner, ttl), mr
}

func TestCachedReader_MissThenHit_OnlyOneInnerCall(t *testing.T) {
	inner := newStub(true, nil)
	cache, _ := newTestCache(t, inner, 60*time.Second)
	userID := uuid.New()

	// First call: miss → delegates.
	active, err := cache.IsActive(context.Background(), userID)
	require.NoError(t, err)
	assert.True(t, active)

	// Second call: hit → no new inner call.
	active2, err := cache.IsActive(context.Background(), userID)
	require.NoError(t, err)
	assert.True(t, active2)

	assert.Equal(t, 1, inner.callsFor[userID], "subsequent calls MUST come from cache")
}

func TestCachedReader_CachesFalseAsWellAsTrue(t *testing.T) {
	// A free user's "false" answer is cached too — otherwise every
	// free user's milestone release would hit the DB on every call.
	inner := newStub(false, nil)
	cache, _ := newTestCache(t, inner, 60*time.Second)
	userID := uuid.New()

	for i := 0; i < 3; i++ {
		active, err := cache.IsActive(context.Background(), userID)
		require.NoError(t, err)
		assert.False(t, active)
	}
	assert.Equal(t, 1, inner.callsFor[userID])
}

func TestCachedReader_ExpiresAfterTTL(t *testing.T) {
	inner := newStub(true, nil)
	cache, mr := newTestCache(t, inner, 60*time.Second)
	userID := uuid.New()

	_, _ = cache.IsActive(context.Background(), userID)

	// Advance miniredis clock past the TTL.
	mr.FastForward(61 * time.Second)

	_, _ = cache.IsActive(context.Background(), userID)
	assert.Equal(t, 2, inner.callsFor[userID], "entry must expire after TTL")
}

func TestCachedReader_InnerError_FailsClosedNoCacheWrite(t *testing.T) {
	inner := newStub(false, errors.New("db lost"))
	cache, mr := newTestCache(t, inner, 60*time.Second)
	userID := uuid.New()

	active, err := cache.IsActive(context.Background(), userID)

	require.Error(t, err, "inner error must surface")
	assert.False(t, active, "MUST fail closed (no Premium on error)")

	// The entry must NOT be cached — a transient DB blip must not pin
	// a free answer for the full TTL.
	_, getErr := mr.Get("sub:active:" + userID.String())
	assert.Equal(t, miniredis.ErrKeyNotFound, getErr)
}

func TestCachedReader_Invalidate_ClearsEntry(t *testing.T) {
	inner := newStub(true, nil)
	cache, mr := newTestCache(t, inner, 60*time.Second)
	userID := uuid.New()

	// Prime cache.
	_, _ = cache.IsActive(context.Background(), userID)

	// Flip the inner answer. Cache still serves the stale true.
	inner.active = false
	active, _ := cache.IsActive(context.Background(), userID)
	assert.True(t, active, "stale hit expected before invalidation")

	require.NoError(t, cache.Invalidate(context.Background(), userID))

	_, getErr := mr.Get("sub:active:" + userID.String())
	assert.Equal(t, miniredis.ErrKeyNotFound, getErr, "invalidated entry must be gone")

	active2, _ := cache.IsActive(context.Background(), userID)
	assert.False(t, active2, "fresh read after invalidation must reflect new inner state")
}

func TestCachedReader_Invalidate_MissingKey_NoError(t *testing.T) {
	inner := newStub(true, nil)
	cache, _ := newTestCache(t, inner, 60*time.Second)

	err := cache.Invalidate(context.Background(), uuid.New())

	assert.NoError(t, err, "Del on a missing key must be a no-op")
}

func TestCachedReader_RedisDown_FallsThroughToInner(t *testing.T) {
	// Point the client at a port nothing is listening on. Redis calls
	// will fail with connection refused / dial timeout. The cache MUST
	// still answer by delegating to inner — the whole point of
	// fail-through is to survive a dead Redis without downtime.
	inner := newStub(true, nil)
	client := goredis.NewClient(&goredis.Options{
		Addr:        "127.0.0.1:1", // reserved port: nothing listens there
		DialTimeout: 150 * time.Millisecond,
	})
	t.Cleanup(func() { _ = client.Close() })

	cache := adapter.NewCachedSubscriptionReader(client, inner, 60*time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	active, err := cache.IsActive(ctx, uuid.New())
	require.NoError(t, err, "Redis down must not fail the request")
	assert.True(t, active, "inner value must be returned")
}

func TestCachedReader_ZeroTTLFallsBackToDefault(t *testing.T) {
	inner := newStub(true, nil)
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	cache := adapter.NewCachedSubscriptionReader(client, inner, 0) // zero → default

	userID := uuid.New()
	_, _ = cache.IsActive(context.Background(), userID)
	ttl := mr.TTL("sub:active:" + userID.String())
	assert.Greater(t, ttl, time.Duration(0), "zero TTL input MUST be normalised to a positive default")
}
