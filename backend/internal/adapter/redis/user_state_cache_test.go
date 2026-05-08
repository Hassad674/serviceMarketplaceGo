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
	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/handler/middleware"
)

// stubStateChecker is a controllable middleware.UserStateChecker used
// to assert the cache's delegation behaviour and count inner calls per
// user id.
type stubStateChecker struct {
	state    middleware.UserState
	err      error
	callsFor map[uuid.UUID]int
}

func newStubChecker(state middleware.UserState, err error) *stubStateChecker {
	return &stubStateChecker{
		state:    state,
		err:      err,
		callsFor: make(map[uuid.UUID]int),
	}
}

func (s *stubStateChecker) GetUserState(_ context.Context, userID uuid.UUID) (middleware.UserState, error) {
	s.callsFor[userID]++
	return s.state, s.err
}

func newTestUserStateCache(
	t *testing.T,
	inner *stubStateChecker,
	ttl time.Duration,
) (*adapter.CachedUserStateChecker, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	return adapter.NewCachedUserStateChecker(client, inner, ttl), mr
}

func TestUserStateCache_MissThenHit_OneInnerCall(t *testing.T) {
	inner := newStubChecker(middleware.UserState{IsAdmin: true, Status: user.StatusActive}, nil)
	cache, _ := newTestUserStateCache(t, inner, 30*time.Second)
	userID := uuid.New()

	// 1st call: miss → delegate + write-through.
	state, err := cache.GetUserState(context.Background(), userID)
	require.NoError(t, err)
	assert.True(t, state.IsAdmin)
	assert.Equal(t, user.StatusActive, state.Status)

	// 2nd, 3rd, 4th calls: hits.
	for i := 0; i < 3; i++ {
		state, err = cache.GetUserState(context.Background(), userID)
		require.NoError(t, err)
		assert.True(t, state.IsAdmin)
	}

	assert.Equal(t, 1, inner.callsFor[userID],
		"subsequent reads MUST come from cache, not the inner checker")
}

func TestUserStateCache_CachesNonAdminAlso(t *testing.T) {
	// IsAdmin=false is the most common state — it MUST also be cached
	// or the entire population's auth path hits the DB on every call.
	inner := newStubChecker(middleware.UserState{IsAdmin: false, Status: user.StatusActive}, nil)
	cache, _ := newTestUserStateCache(t, inner, 30*time.Second)
	userID := uuid.New()

	for i := 0; i < 5; i++ {
		state, err := cache.GetUserState(context.Background(), userID)
		require.NoError(t, err)
		assert.False(t, state.IsAdmin)
	}
	assert.Equal(t, 1, inner.callsFor[userID])
}

func TestUserStateCache_TTLExpiresThenRefetches(t *testing.T) {
	inner := newStubChecker(middleware.UserState{IsAdmin: false, Status: user.StatusActive}, nil)
	cache, mr := newTestUserStateCache(t, inner, 30*time.Second)
	userID := uuid.New()

	_, err := cache.GetUserState(context.Background(), userID)
	require.NoError(t, err)
	assert.Equal(t, 1, inner.callsFor[userID])

	// Fast-forward miniredis past the TTL → next call must miss again.
	mr.FastForward(31 * time.Second)

	_, err = cache.GetUserState(context.Background(), userID)
	require.NoError(t, err)
	assert.Equal(t, 2, inner.callsFor[userID],
		"after TTL expiry, the cache MUST consult the inner checker again")
}

func TestUserStateCache_Invalidate_EvictsImmediately(t *testing.T) {
	// Promote-then-read flow. The first GetUserState caches IsAdmin=false.
	// After Invalidate, the next read MUST re-consult the inner checker
	// (which now returns IsAdmin=true) so the new state surfaces
	// without waiting for the TTL.
	state := middleware.UserState{IsAdmin: false, Status: user.StatusActive}
	inner := &stubStateChecker{
		state:    state,
		callsFor: make(map[uuid.UUID]int),
	}
	cache, _ := newTestUserStateCache(t, inner, 30*time.Second)
	userID := uuid.New()

	got, err := cache.GetUserState(context.Background(), userID)
	require.NoError(t, err)
	assert.False(t, got.IsAdmin)

	// Operator promotion: bump the inner answer + invalidate.
	inner.state = middleware.UserState{IsAdmin: true, Status: user.StatusActive}
	require.NoError(t, cache.Invalidate(context.Background(), userID))

	got, err = cache.GetUserState(context.Background(), userID)
	require.NoError(t, err)
	assert.True(t, got.IsAdmin, "invalidate MUST evict so the next read sees the live value")
	assert.Equal(t, 2, inner.callsFor[userID], "exactly two inner reads — one before, one after")
}

func TestUserStateCache_InnerError_NotCached(t *testing.T) {
	// Lookup error must NOT be cached as a negative answer — that
	// would amplify a transient blip into a 30s lockout.
	inner := newStubChecker(middleware.UserState{}, errors.New("postgres: connection refused"))
	cache, _ := newTestUserStateCache(t, inner, 30*time.Second)
	userID := uuid.New()

	for i := 0; i < 3; i++ {
		_, err := cache.GetUserState(context.Background(), userID)
		assert.Error(t, err)
	}
	assert.Equal(t, 3, inner.callsFor[userID],
		"errors MUST NOT be cached — every call goes through to the inner checker")
}

func TestUserStateCache_UserNotFound_PropagatesNotCached(t *testing.T) {
	// ErrUserNotFound is a domain sentinel: middleware turns it into a
	// 401 (zombie session). Same negative-cache rule applies — the
	// inner reader is the source of truth.
	inner := newStubChecker(middleware.UserState{}, user.ErrUserNotFound)
	cache, _ := newTestUserStateCache(t, inner, 30*time.Second)
	userID := uuid.New()

	for i := 0; i < 2; i++ {
		_, err := cache.GetUserState(context.Background(), userID)
		assert.ErrorIs(t, err, user.ErrUserNotFound)
	}
	assert.Equal(t, 2, inner.callsFor[userID])
}

func TestUserStateCache_RedisDown_FallsThroughToInner(t *testing.T) {
	// Close the underlying client so every Redis op errors. The cache
	// must still surface the inner answer — a degraded cache MUST NOT
	// take the request path down with it.
	mr, err := miniredis.Run()
	require.NoError(t, err)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	mr.Close() // simulate Redis going away

	inner := newStubChecker(middleware.UserState{IsAdmin: true, Status: user.StatusActive}, nil)
	cache := adapter.NewCachedUserStateChecker(client, inner, 30*time.Second)

	state, err := cache.GetUserState(context.Background(), uuid.New())
	require.NoError(t, err, "Redis outage MUST NOT propagate as an error")
	assert.True(t, state.IsAdmin)
}

func TestUserStateCache_DefaultTTL_AppliedWhenZero(t *testing.T) {
	// Constructor convenience: passing 0 must fall back to the
	// documented default rather than write entries with no expiry.
	inner := newStubChecker(middleware.UserState{Status: user.StatusActive}, nil)
	cache, mr := newTestUserStateCache(t, inner, 0)
	userID := uuid.New()

	_, err := cache.GetUserState(context.Background(), userID)
	require.NoError(t, err)

	keys := mr.Keys()
	require.Len(t, keys, 1, "exactly one cache key written")
	ttl := mr.TTL(keys[0])
	assert.True(t, ttl > 0, "default TTL must be positive")
	assert.True(t, ttl <= adapter.DefaultUserStateCacheTTL,
		"default TTL must be at most the documented %s, got %s",
		adapter.DefaultUserStateCacheTTL, ttl)
}
