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
)

// stubVersionChecker is a controllable middleware.SessionVersionChecker
// used to assert the cache's delegation behaviour and count inner
// calls per user id.
type stubVersionChecker struct {
	version  int
	err      error
	callsFor map[uuid.UUID]int
}

func newStubVersionChecker(version int, err error) *stubVersionChecker {
	return &stubVersionChecker{
		version:  version,
		err:      err,
		callsFor: make(map[uuid.UUID]int),
	}
}

func (s *stubVersionChecker) GetSessionVersion(_ context.Context, userID uuid.UUID) (int, error) {
	s.callsFor[userID]++
	return s.version, s.err
}

func newTestSessionVersionCache(
	t *testing.T,
	inner *stubVersionChecker,
	ttl time.Duration,
) (*adapter.CachedSessionVersionChecker, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	cache := adapter.NewCachedSessionVersionChecker(client, inner, ttl)
	return cache, mr
}

func TestSessionVersionCache_MissThenHit(t *testing.T) {
	t.Parallel()
	inner := newStubVersionChecker(7, nil)
	cache, _ := newTestSessionVersionCache(t, inner, 30*time.Second)

	uid := uuid.New()
	ctx := context.Background()

	// First call → miss → delegate.
	got, err := cache.GetSessionVersion(ctx, uid)
	require.NoError(t, err)
	assert.Equal(t, 7, got)
	assert.Equal(t, 1, inner.callsFor[uid], "inner called on first miss")

	// Second call → hit → no inner call.
	got, err = cache.GetSessionVersion(ctx, uid)
	require.NoError(t, err)
	assert.Equal(t, 7, got)
	assert.Equal(t, 1, inner.callsFor[uid], "inner NOT re-called on cache hit")
}

func TestSessionVersionCache_TTLExpires(t *testing.T) {
	t.Parallel()
	inner := newStubVersionChecker(3, nil)
	cache, mr := newTestSessionVersionCache(t, inner, 5*time.Second)

	uid := uuid.New()
	ctx := context.Background()

	_, err := cache.GetSessionVersion(ctx, uid)
	require.NoError(t, err)
	assert.Equal(t, 1, inner.callsFor[uid])

	// Fast-forward Redis past the TTL.
	mr.FastForward(6 * time.Second)

	_, err = cache.GetSessionVersion(ctx, uid)
	require.NoError(t, err)
	assert.Equal(t, 2, inner.callsFor[uid], "inner re-called after TTL expiry")
}

func TestSessionVersionCache_DefaultTTLAppliedWhenZero(t *testing.T) {
	t.Parallel()
	inner := newStubVersionChecker(1, nil)
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	// Pass 0 — must fall back to DefaultSessionVersionCacheTTL.
	cache := adapter.NewCachedSessionVersionChecker(client, inner, 0)
	uid := uuid.New()
	ctx := context.Background()

	_, err = cache.GetSessionVersion(ctx, uid)
	require.NoError(t, err)

	ttl := mr.TTL("session_version:" + uid.String())
	assert.InDelta(t, adapter.DefaultSessionVersionCacheTTL.Seconds(), ttl.Seconds(), 1.0)
}

func TestSessionVersionCache_DoesNotCacheUserNotFound(t *testing.T) {
	t.Parallel()
	inner := newStubVersionChecker(0, user.ErrUserNotFound)
	cache, mr := newTestSessionVersionCache(t, inner, 30*time.Second)

	uid := uuid.New()
	ctx := context.Background()

	_, err := cache.GetSessionVersion(ctx, uid)
	assert.ErrorIs(t, err, user.ErrUserNotFound)
	// Make sure no key was written for an error response.
	_, redisErr := mr.Get("session_version:" + uid.String())
	assert.ErrorIs(t, redisErr, miniredis.ErrKeyNotFound)
	assert.Equal(t, 1, inner.callsFor[uid])

	// Calling again must hit the inner (no negative caching).
	_, _ = cache.GetSessionVersion(ctx, uid)
	assert.Equal(t, 2, inner.callsFor[uid])
}

func TestSessionVersionCache_TransientInnerErrorNotCached(t *testing.T) {
	t.Parallel()
	transient := errors.New("postgres unreachable")
	inner := newStubVersionChecker(0, transient)
	cache, mr := newTestSessionVersionCache(t, inner, 30*time.Second)

	uid := uuid.New()
	ctx := context.Background()

	_, err := cache.GetSessionVersion(ctx, uid)
	assert.ErrorIs(t, err, transient)

	_, redisErr := mr.Get("session_version:" + uid.String())
	assert.ErrorIs(t, redisErr, miniredis.ErrKeyNotFound, "errors must NEVER be cached")
}

func TestSessionVersionCache_MalformedPayloadRefreshes(t *testing.T) {
	t.Parallel()
	inner := newStubVersionChecker(11, nil)
	cache, mr := newTestSessionVersionCache(t, inner, 30*time.Second)

	uid := uuid.New()
	// Plant a malformed payload directly.
	require.NoError(t, mr.Set("session_version:"+uid.String(), "not-a-number"))

	got, err := cache.GetSessionVersion(context.Background(), uid)
	require.NoError(t, err)
	assert.Equal(t, 11, got)
	assert.Equal(t, 1, inner.callsFor[uid], "inner called when payload corrupt")

	// The write-through should have replaced the malformed entry.
	val, getErr := mr.Get("session_version:" + uid.String())
	require.NoError(t, getErr)
	assert.Equal(t, "11", val)
}

func TestSessionVersionCache_Invalidate(t *testing.T) {
	t.Parallel()
	inner := newStubVersionChecker(5, nil)
	cache, mr := newTestSessionVersionCache(t, inner, 30*time.Second)
	uid := uuid.New()
	ctx := context.Background()

	// Prime the cache.
	_, err := cache.GetSessionVersion(ctx, uid)
	require.NoError(t, err)

	val, err := mr.Get("session_version:" + uid.String())
	require.NoError(t, err)
	require.Equal(t, "5", val)

	// Invalidate.
	require.NoError(t, cache.Invalidate(ctx, uid))
	_, err = mr.Get("session_version:" + uid.String())
	assert.ErrorIs(t, err, miniredis.ErrKeyNotFound)

	// Missing key is NOT an error.
	require.NoError(t, cache.Invalidate(ctx, uuid.New()))
}

func TestSessionVersionCache_ReturnsZeroFromInner(t *testing.T) {
	// A brand-new user with session_version=0 must still be cached
	// correctly (zero is a legitimate value).
	t.Parallel()
	inner := newStubVersionChecker(0, nil)
	cache, _ := newTestSessionVersionCache(t, inner, 30*time.Second)
	uid := uuid.New()
	ctx := context.Background()

	got, err := cache.GetSessionVersion(ctx, uid)
	require.NoError(t, err)
	assert.Equal(t, 0, got)
	// Hit: second call uses the cached zero.
	got, err = cache.GetSessionVersion(ctx, uid)
	require.NoError(t, err)
	assert.Equal(t, 0, got)
	assert.Equal(t, 1, inner.callsFor[uid])
}
