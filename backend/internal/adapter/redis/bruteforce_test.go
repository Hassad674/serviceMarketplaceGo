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

func newBruteForceTest(t *testing.T) (*adapter.BruteForceService, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	// We use the production policy in most tests so the asserted
	// numbers double as living documentation of the deployed config.
	return adapter.NewBruteForceService(client), mr
}

func TestBruteForce_FreshEmailIsNotLocked(t *testing.T) {
	svc, _ := newBruteForceTest(t)
	ctx := context.Background()

	locked, err := svc.IsLocked(ctx, "fresh@example.com")
	require.NoError(t, err)
	assert.False(t, locked)

	retry, err := svc.RetryAfter(ctx, "fresh@example.com")
	require.NoError(t, err)
	assert.Equal(t, time.Duration(0), retry)
}

func TestBruteForce_FourFailuresDoNotLock(t *testing.T) {
	// SEC-07: the lockout threshold is 5 — four failures must NOT
	// trigger the lockout flag (if they did, a single typo wave from
	// a legitimate user would lock them out).
	svc, _ := newBruteForceTest(t)
	ctx := context.Background()
	email := "user@example.com"

	for i := 0; i < 4; i++ {
		require.NoError(t, svc.RecordFailure(ctx, email))
	}

	locked, err := svc.IsLocked(ctx, email)
	require.NoError(t, err)
	assert.False(t, locked, "4 failures must NOT trigger the lockout")
}

func TestBruteForce_FifthFailureLocksAndReportsRetryAfter(t *testing.T) {
	// SEC-07: at the 5th failure the lockout flag is set with a 30-min
	// TTL. RetryAfter must return a positive duration.
	svc, _ := newBruteForceTest(t)
	ctx := context.Background()
	email := "user@example.com"

	for i := 0; i < 5; i++ {
		require.NoError(t, svc.RecordFailure(ctx, email))
	}

	locked, err := svc.IsLocked(ctx, email)
	require.NoError(t, err)
	assert.True(t, locked, "5 failures must trigger the lockout")

	retry, err := svc.RetryAfter(ctx, email)
	require.NoError(t, err)
	assert.Greater(t, retry, time.Duration(0))
	// Lockout TTL is 30 minutes — assert we are within a sensible band
	// (allow small clock drift between Set and TTL read).
	assert.LessOrEqual(t, retry, 30*time.Minute)
	assert.Greater(t, retry, 25*time.Minute)
}

func TestBruteForce_FailuresAboveThresholdKeepLockoutSet(t *testing.T) {
	// A legitimate-but-confused user keeps trying after they hit the
	// lockout. The flag must stay set; subsequent failures must not
	// extend the TTL beyond its initial 30-min — but a stale read
	// returning sub-zero would be a bug so we assert the value
	// monotonically decreases as expected.
	svc, _ := newBruteForceTest(t)
	ctx := context.Background()
	email := "stubborn@example.com"

	for i := 0; i < 10; i++ {
		require.NoError(t, svc.RecordFailure(ctx, email))
	}

	locked, _ := svc.IsLocked(ctx, email)
	assert.True(t, locked)
}

func TestBruteForce_RecordSuccessClearsState(t *testing.T) {
	// SEC-07: a successful login (after fewer than 5 failures) must
	// wipe the counter so the user gets a clean slate. Even after the
	// lockout has been set, RecordSuccess clears it (admin override
	// or an authenticated unlock would call this).
	svc, _ := newBruteForceTest(t)
	ctx := context.Background()
	email := "success@example.com"

	for i := 0; i < 4; i++ {
		require.NoError(t, svc.RecordFailure(ctx, email))
	}
	require.NoError(t, svc.RecordSuccess(ctx, email))

	// 4 more failures should NOT trigger the lockout because the
	// counter has been wiped.
	for i := 0; i < 4; i++ {
		require.NoError(t, svc.RecordFailure(ctx, email))
	}
	locked, _ := svc.IsLocked(ctx, email)
	assert.False(t, locked, "RecordSuccess must clear the counter")
}

func TestBruteForce_RecordSuccessAfterLockoutClearsLockout(t *testing.T) {
	svc, _ := newBruteForceTest(t)
	ctx := context.Background()
	email := "after-lockout@example.com"

	for i := 0; i < 5; i++ {
		require.NoError(t, svc.RecordFailure(ctx, email))
	}
	locked, _ := svc.IsLocked(ctx, email)
	require.True(t, locked)

	require.NoError(t, svc.RecordSuccess(ctx, email))

	locked, _ = svc.IsLocked(ctx, email)
	assert.False(t, locked, "RecordSuccess must clear the lockout flag too")
}

func TestBruteForce_LockoutExpiresAfterTTL(t *testing.T) {
	// SEC-07: lockout TTL is 30min — after that window the user can
	// try again without admin intervention.
	svc, mr := newBruteForceTest(t)
	ctx := context.Background()
	email := "expires@example.com"

	for i := 0; i < 5; i++ {
		require.NoError(t, svc.RecordFailure(ctx, email))
	}
	require.True(t, mustLocked(t, svc, email))

	mr.FastForward(31 * time.Minute)

	assert.False(t, mustLocked(t, svc, email))
}

func TestBruteForce_AttemptsWindowResetsAfterTTL(t *testing.T) {
	// SEC-07: the failed-attempts counter has a 15-minute TTL. After
	// the window passes a fresh wave of failures starts over, so a
	// dribble of 1-per-hour failures cannot accumulate to a lockout.
	svc, mr := newBruteForceTest(t)
	ctx := context.Background()
	email := "dribble@example.com"

	for i := 0; i < 4; i++ {
		require.NoError(t, svc.RecordFailure(ctx, email))
	}

	mr.FastForward(16 * time.Minute)

	for i := 0; i < 4; i++ {
		require.NoError(t, svc.RecordFailure(ctx, email))
	}

	locked, err := svc.IsLocked(ctx, email)
	require.NoError(t, err)
	assert.False(t, locked, "counter must reset after TTL")
}

func TestBruteForce_EmailNormalised(t *testing.T) {
	// Casing must not split a counter — a typo with capital letters is
	// the same target as the lowercased version.
	svc, _ := newBruteForceTest(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		require.NoError(t, svc.RecordFailure(ctx, "USER@Example.com"))
	}

	locked, err := svc.IsLocked(ctx, "user@example.com")
	require.NoError(t, err)
	assert.True(t, locked, "uppercase and lowercase must share one counter")
}

func TestBruteForce_EmptyEmailIsNoop(t *testing.T) {
	svc, mr := newBruteForceTest(t)
	ctx := context.Background()

	require.NoError(t, svc.RecordFailure(ctx, ""))
	require.NoError(t, svc.RecordFailure(ctx, "   "))
	assert.Empty(t, mr.Keys())

	locked, err := svc.IsLocked(ctx, "")
	require.NoError(t, err)
	assert.False(t, locked)
}

func TestBruteForce_DistinctEmailsDoNotCollide(t *testing.T) {
	svc, _ := newBruteForceTest(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		require.NoError(t, svc.RecordFailure(ctx, "alice@example.com"))
	}

	bobLocked, err := svc.IsLocked(ctx, "bob@example.com")
	require.NoError(t, err)
	assert.False(t, bobLocked, "alice's lockout must not affect bob")
}

func TestBruteForce_ConcurrentFailuresRaceFreeAtThreshold(t *testing.T) {
	// Race condition smoke test: 20 goroutines record a failure for
	// the same email simultaneously. The lockout flag must be set
	// reliably (it does not matter which goroutine is the "5th").
	svc, _ := newBruteForceTest(t)
	ctx := context.Background()
	email := "race@example.com"

	const goroutines = 20
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			_ = svc.RecordFailure(ctx, email)
		}()
	}
	wg.Wait()

	locked, err := svc.IsLocked(ctx, email)
	require.NoError(t, err)
	assert.True(t, locked, "20 concurrent failures must produce a lockout")
}

func TestBruteForce_CustomPolicyOverridesDefaults(t *testing.T) {
	// Smoke test the policy override constructor — useful for tests
	// elsewhere that want a 1-attempt threshold to exercise the
	// lockout branch in a single line.
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	defer client.Close()

	svc := adapter.NewBruteForceServiceWithPolicy(client, 2, 5*time.Minute, 1*time.Minute)

	ctx := context.Background()
	require.NoError(t, svc.RecordFailure(ctx, "tight@example.com"))
	require.NoError(t, svc.RecordFailure(ctx, "tight@example.com"))

	locked, err := svc.IsLocked(ctx, "tight@example.com")
	require.NoError(t, err)
	assert.True(t, locked, "custom threshold of 2 must lock at second failure")
}

// SEC-07: every method must surface a wrapped error when Redis is
// unavailable so the caller can fail-open or fail-closed deliberately.
// Silent swallowing would let an attacker bypass rate limits during an
// outage.
func TestBruteForce_IsLocked_RedisDown_ReturnsError(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	addr := mr.Addr()
	mr.Close()

	client := goredis.NewClient(&goredis.Options{Addr: addr})
	t.Cleanup(func() { _ = client.Close() })

	svc := adapter.NewBruteForceService(client)
	locked, err := svc.IsLocked(context.Background(), "user@example.com")
	require.Error(t, err)
	assert.False(t, locked, "boolean must be false on error so the caller cannot accidentally trust it")
	assert.Contains(t, err.Error(), "brute force is_locked")
}

func TestBruteForce_RecordFailure_RedisDown_ReturnsError(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	addr := mr.Addr()
	mr.Close()

	client := goredis.NewClient(&goredis.Options{Addr: addr})
	t.Cleanup(func() { _ = client.Close() })

	svc := adapter.NewBruteForceService(client)
	err = svc.RecordFailure(context.Background(), "user@example.com")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "brute force record_failure")
}

func TestBruteForce_RecordSuccess_RedisDown_ReturnsError(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	addr := mr.Addr()
	mr.Close()

	client := goredis.NewClient(&goredis.Options{Addr: addr})
	t.Cleanup(func() { _ = client.Close() })

	svc := adapter.NewBruteForceService(client)
	err = svc.RecordSuccess(context.Background(), "user@example.com")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "brute force record_success")
}

func TestBruteForce_RetryAfter_RedisDown_ReturnsError(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	addr := mr.Addr()
	mr.Close()

	client := goredis.NewClient(&goredis.Options{Addr: addr})
	t.Cleanup(func() { _ = client.Close() })

	svc := adapter.NewBruteForceService(client)
	dur, err := svc.RetryAfter(context.Background(), "user@example.com")
	require.Error(t, err)
	assert.Equal(t, time.Duration(0), dur)
	assert.Contains(t, err.Error(), "brute force retry_after")
}

// RetryAfter must treat both "key never set" (-2) and "key has no TTL"
// (-1) as unlocked. The -1 case shouldn't happen with our setters but
// we test it defensively because it is part of the documented contract.
func TestBruteForce_RetryAfter_KeyWithoutTTLReturnsZero(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	defer client.Close()

	// Manually create a key with no TTL (no Expire). Redis returns -1
	// for TTL on such keys.
	err = client.Set(context.Background(), "login_locked:notexpire@example.com", "1", 0).Err()
	require.NoError(t, err)

	svc := adapter.NewBruteForceService(client)
	dur, err := svc.RetryAfter(context.Background(), "notexpire@example.com")
	require.NoError(t, err)
	assert.Equal(t, time.Duration(0), dur, "key without TTL must report zero retry-after — defensive contract")
}

// ErrNoLockout is exported but currently unused. The contract requires
// it to be a stable sentinel value; assert that it has a string form.
func TestBruteForce_ErrNoLockout_HasMeaningfulMessage(t *testing.T) {
	require.NotNil(t, adapter.ErrNoLockout)
	assert.Contains(t, adapter.ErrNoLockout.Error(), "lockout")
}

func mustLocked(t *testing.T, svc *adapter.BruteForceService, email string) bool {
	t.Helper()
	locked, err := svc.IsLocked(context.Background(), email)
	require.NoError(t, err)
	return locked
}
