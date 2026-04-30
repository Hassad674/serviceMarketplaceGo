package redis_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	adapter "marketplace-backend/internal/adapter/redis"
)

// newRefreshBlacklistTest spins up a miniredis instance bound to a
// fresh go-redis client and returns the adapter under test plus the
// miniredis handle so the test can FastForward time without sleeping.
func newRefreshBlacklistTest(t *testing.T) (*adapter.RefreshBlacklistService, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	return adapter.NewRefreshBlacklistService(client), mr
}

func TestRefreshBlacklist_AddThenHasReturnsTrue(t *testing.T) {
	svc, _ := newRefreshBlacklistTest(t)
	ctx := context.Background()

	require.NoError(t, svc.Add(ctx, "jti-abc", time.Hour))

	has, err := svc.Has(ctx, "jti-abc")
	require.NoError(t, err)
	assert.True(t, has, "blacklisted jti must report true")
}

func TestRefreshBlacklist_FreshJTIIsNotBlacklisted(t *testing.T) {
	svc, _ := newRefreshBlacklistTest(t)
	ctx := context.Background()

	has, err := svc.Has(ctx, "never-blacklisted")
	require.NoError(t, err)
	assert.False(t, has)
}

func TestRefreshBlacklist_TTLExpiresAfterDuration(t *testing.T) {
	// SEC-06: the blacklist entry's TTL must match the original
	// token's remaining time-to-expire — once that window passes, the
	// entry can disappear because the underlying token would have
	// failed validation anyway.
	svc, mr := newRefreshBlacklistTest(t)
	ctx := context.Background()

	require.NoError(t, svc.Add(ctx, "jti-short", 1*time.Minute))

	mr.FastForward(2 * time.Minute)

	has, err := svc.Has(ctx, "jti-short")
	require.NoError(t, err)
	assert.False(t, has, "expired blacklist entry must not match")
}

func TestRefreshBlacklist_EmptyJTIIsNoop(t *testing.T) {
	// Defensive: an empty jti must be a no-op so handlers do not need
	// a defensive check before delegating to the blacklist.
	svc, mr := newRefreshBlacklistTest(t)
	ctx := context.Background()

	require.NoError(t, svc.Add(ctx, "", time.Hour))
	assert.Empty(t, mr.Keys(), "empty jti must not write any key")

	has, err := svc.Has(ctx, "")
	require.NoError(t, err)
	assert.False(t, has)
}

func TestRefreshBlacklist_NegativeTTLIsNoop(t *testing.T) {
	// A token already past its expiry has nothing to gain from being
	// blacklisted — it would fail validation anyway. Add() must not
	// silently set an entry with a TTL of 0 (which Redis would treat
	// as "expire immediately" and may even keep momentarily).
	svc, mr := newRefreshBlacklistTest(t)
	ctx := context.Background()

	require.NoError(t, svc.Add(ctx, "jti-expired", -1*time.Second))
	assert.Empty(t, mr.Keys())

	require.NoError(t, svc.Add(ctx, "jti-zero", 0))
	assert.Empty(t, mr.Keys())
}

func TestRefreshBlacklist_DistinctJTIsDoNotCollide(t *testing.T) {
	svc, _ := newRefreshBlacklistTest(t)
	ctx := context.Background()

	require.NoError(t, svc.Add(ctx, "jti-a", time.Hour))

	hasA, _ := svc.Has(ctx, "jti-a")
	hasB, _ := svc.Has(ctx, "jti-b")

	assert.True(t, hasA)
	assert.False(t, hasB, "an unrelated jti must not be blacklisted")
}

func TestRefreshBlacklist_ReAddRefreshesTTL(t *testing.T) {
	// In the rare case a JTI is rotated twice (e.g. parallel logout +
	// refresh races), the second Add must overwrite the first with
	// the new TTL — there is no scenario where we want to keep the
	// older, shorter TTL.
	svc, mr := newRefreshBlacklistTest(t)
	ctx := context.Background()

	require.NoError(t, svc.Add(ctx, "jti-rot", 1*time.Minute))
	require.NoError(t, svc.Add(ctx, "jti-rot", 10*time.Minute))

	mr.FastForward(5 * time.Minute) // past the first TTL
	has, err := svc.Has(ctx, "jti-rot")
	require.NoError(t, err)
	assert.True(t, has, "second Add must extend the TTL")
}

// SEC-06: when Redis is unavailable Add must surface a wrapped error
// so the caller can either fail closed (security-sensitive logout) or
// fail open (less critical paths) — the helper must NOT silently
// swallow the failure.
func TestRefreshBlacklist_Add_RedisDown_ReturnsError(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	addr := mr.Addr()
	mr.Close()

	client := goredis.NewClient(&goredis.Options{Addr: addr})
	t.Cleanup(func() { _ = client.Close() })

	svc := adapter.NewRefreshBlacklistService(client)
	err = svc.Add(context.Background(), "jti-down", time.Hour)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "refresh blacklist add")
}

// Has must surface a wrapped error so the caller (auth refresh) does
// not mistake a Redis blip for a missing blacklist entry — that would
// allow a stolen refresh token to keep working through outages.
func TestRefreshBlacklist_Has_RedisDown_ReturnsError(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	addr := mr.Addr()
	mr.Close()

	client := goredis.NewClient(&goredis.Options{Addr: addr})
	t.Cleanup(func() { _ = client.Close() })

	svc := adapter.NewRefreshBlacklistService(client)
	has, err := svc.Has(context.Background(), "jti-down")
	require.Error(t, err)
	assert.False(t, has, "the boolean must be false on error so the caller cannot accidentally trust the result")
	assert.Contains(t, err.Error(), "refresh blacklist has")
}

func TestRefreshBlacklist_ConcurrentAddsAreSafe(t *testing.T) {
	// Race condition smoke test: 10 goroutines blacklist the same JTI
	// at once. The end state must be a single, valid blacklist entry —
	// no panic, no inconsistent visibility from Has().
	svc, _ := newRefreshBlacklistTest(t)
	ctx := context.Background()

	const goroutines = 10
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			_ = svc.Add(ctx, "jti-concurrent", time.Hour)
		}()
	}
	wg.Wait()

	has, err := svc.Has(ctx, "jti-concurrent")
	require.NoError(t, err)
	assert.True(t, has)
}
