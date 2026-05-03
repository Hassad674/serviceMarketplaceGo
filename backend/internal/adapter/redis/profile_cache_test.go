package redis_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	adapter "marketplace-backend/internal/adapter/redis"
	"marketplace-backend/internal/domain/profile"
)

// stubProfileReader is a controllable PublicProfileReader used to
// assert the cache's delegation behaviour and count inner calls.
type stubProfileReader struct {
	mu    sync.Mutex
	p     *profile.Profile
	err   error
	calls map[uuid.UUID]int
	delay time.Duration
}

func newStubProfile(p *profile.Profile, err error) *stubProfileReader {
	return &stubProfileReader{
		p:     p,
		err:   err,
		calls: make(map[uuid.UUID]int),
	}
}

func (s *stubProfileReader) GetProfile(_ context.Context, orgID uuid.UUID) (*profile.Profile, error) {
	s.mu.Lock()
	s.calls[orgID]++
	d := s.delay
	s.mu.Unlock()
	if d > 0 {
		time.Sleep(d)
	}
	return s.p, s.err
}

func (s *stubProfileReader) callCount(orgID uuid.UUID) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls[orgID]
}

func newProfileTestCache(t *testing.T, inner *stubProfileReader, ttl, negTTL time.Duration) (*adapter.CachedPublicProfileReader, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	return adapter.NewCachedPublicProfileReader(client, inner, ttl, negTTL), mr
}

func samplePresentProfile(orgID uuid.UUID) *profile.Profile {
	return &profile.Profile{
		OrganizationID:        orgID,
		Title:                 "Senior Backend Engineer",
		About:                 "Hello world",
		PhotoURL:              "https://cdn.example.com/avatar.png",
		City:                  "Paris",
		CountryCode:           "FR",
		WorkMode:              []string{"remote"},
		LanguagesProfessional: []string{"en", "fr"},
		AvailabilityStatus:    profile.AvailabilityNow,
	}
}

// --- Hit / miss / TTL ---

func TestProfileCache_MissThenHit_OneInnerCall(t *testing.T) {
	orgID := uuid.New()
	want := samplePresentProfile(orgID)
	inner := newStubProfile(want, nil)
	cache, _ := newProfileTestCache(t, inner, 60*time.Second, 30*time.Second)

	first, err := cache.GetProfile(context.Background(), orgID)
	require.NoError(t, err)
	assert.Equal(t, want.Title, first.Title)
	assert.Equal(t, want.City, first.City)
	assert.Equal(t, want.LanguagesProfessional, first.LanguagesProfessional)

	second, err := cache.GetProfile(context.Background(), orgID)
	require.NoError(t, err)
	assert.Equal(t, want.Title, second.Title)

	assert.Equal(t, 1, inner.callCount(orgID), "second call must come from cache")
}

func TestProfileCache_ExpiresAfterTTL(t *testing.T) {
	orgID := uuid.New()
	inner := newStubProfile(samplePresentProfile(orgID), nil)
	cache, mr := newProfileTestCache(t, inner, 60*time.Second, 30*time.Second)

	_, _ = cache.GetProfile(context.Background(), orgID)
	mr.FastForward(61 * time.Second)
	_, _ = cache.GetProfile(context.Background(), orgID)

	assert.Equal(t, 2, inner.callCount(orgID), "entry must expire after TTL")
}

// --- Negative cache ---

func TestProfileCache_NotFound_IsNegativeCached(t *testing.T) {
	// 404 spam scenario: a broken deep-link refreshed a thousand
	// times must hit Redis after the first DB miss.
	orgID := uuid.New()
	inner := newStubProfile(nil, profile.ErrProfileNotFound)
	cache, _ := newProfileTestCache(t, inner, 60*time.Second, 30*time.Second)

	for i := 0; i < 5; i++ {
		_, err := cache.GetProfile(context.Background(), orgID)
		require.ErrorIs(t, err, profile.ErrProfileNotFound)
	}

	assert.Equal(t, 1, inner.callCount(orgID), "negative cache must absorb 404 spam after the first DB miss")
}

func TestProfileCache_NotFound_NegativeTTL_ShorterThanPositive(t *testing.T) {
	// Negative entries should age out faster than positive ones so a
	// missing org that is later created surfaces quickly.
	orgID := uuid.New()
	inner := newStubProfile(nil, profile.ErrProfileNotFound)
	cache, mr := newProfileTestCache(t, inner, 60*time.Second, 30*time.Second)

	_, err := cache.GetProfile(context.Background(), orgID)
	require.ErrorIs(t, err, profile.ErrProfileNotFound)

	// Advance past the negative TTL but well within the positive TTL.
	mr.FastForward(31 * time.Second)
	_, err = cache.GetProfile(context.Background(), orgID)
	require.ErrorIs(t, err, profile.ErrProfileNotFound)

	assert.Equal(t, 2, inner.callCount(orgID), "negative entry must age out after negativeTTL")
}

func TestProfileCache_OtherInnerError_NotCached(t *testing.T) {
	// A transient DB blip (NOT a clean not-found) must NOT pin a 5xx
	// for the full TTL — the next call should retry.
	orgID := uuid.New()
	inner := newStubProfile(nil, errors.New("connection lost"))
	cache, mr := newProfileTestCache(t, inner, 60*time.Second, 30*time.Second)

	_, err := cache.GetProfile(context.Background(), orgID)
	require.Error(t, err)
	require.NotErrorIs(t, err, profile.ErrProfileNotFound)

	_, getErr := mr.Get("profile:agency:" + orgID.String())
	assert.Equal(t, miniredis.ErrKeyNotFound, getErr,
		"transient errors must NOT be cached")
}

// --- Invalidation ---

func TestProfileCache_Invalidate_ClearsHit(t *testing.T) {
	orgID := uuid.New()
	first := samplePresentProfile(orgID)
	first.Title = "First Title"
	inner := newStubProfile(first, nil)
	cache, mr := newProfileTestCache(t, inner, 60*time.Second, 30*time.Second)

	_, _ = cache.GetProfile(context.Background(), orgID)

	// Flip the inner answer; cache still serves stale.
	updated := samplePresentProfile(orgID)
	updated.Title = "Updated Title"
	inner.mu.Lock()
	inner.p = updated
	inner.mu.Unlock()

	stale, _ := cache.GetProfile(context.Background(), orgID)
	assert.Equal(t, "First Title", stale.Title, "stale hit expected before invalidation")

	require.NoError(t, cache.Invalidate(context.Background(), orgID))

	_, getErr := mr.Get("profile:agency:" + orgID.String())
	assert.Equal(t, miniredis.ErrKeyNotFound, getErr)

	fresh, _ := cache.GetProfile(context.Background(), orgID)
	assert.Equal(t, "Updated Title", fresh.Title)
}

func TestProfileCache_Invalidate_ClearsNegativeEntry(t *testing.T) {
	// After an org is created the next read must go through to
	// the DB even though the negative marker is still warm.
	orgID := uuid.New()
	inner := newStubProfile(nil, profile.ErrProfileNotFound)
	cache, _ := newProfileTestCache(t, inner, 60*time.Second, 30*time.Second)

	_, err := cache.GetProfile(context.Background(), orgID)
	require.ErrorIs(t, err, profile.ErrProfileNotFound)

	// Org gets created — the create handler must invalidate.
	created := samplePresentProfile(orgID)
	inner.mu.Lock()
	inner.p = created
	inner.err = nil
	inner.mu.Unlock()

	require.NoError(t, cache.Invalidate(context.Background(), orgID))

	got, err := cache.GetProfile(context.Background(), orgID)
	require.NoError(t, err)
	assert.Equal(t, created.Title, got.Title)
}

func TestProfileCache_Invalidate_MissingKey_NoError(t *testing.T) {
	cache, _ := newProfileTestCache(t, newStubProfile(nil, nil), 60*time.Second, 30*time.Second)
	err := cache.Invalidate(context.Background(), uuid.New())
	assert.NoError(t, err)
}

// --- Resilience ---

func TestProfileCache_RedisDown_FallsThroughToInner(t *testing.T) {
	orgID := uuid.New()
	inner := newStubProfile(samplePresentProfile(orgID), nil)
	client := goredis.NewClient(&goredis.Options{
		Addr:        "127.0.0.1:1",
		DialTimeout: 150 * time.Millisecond,
	})
	t.Cleanup(func() { _ = client.Close() })

	cache := adapter.NewCachedPublicProfileReader(client, inner, 60*time.Second, 30*time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	got, err := cache.GetProfile(ctx, orgID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "Senior Backend Engineer", got.Title)
}

func TestProfileCache_ZeroTTLs_FallBackToDefaults(t *testing.T) {
	orgID := uuid.New()
	inner := newStubProfile(samplePresentProfile(orgID), nil)
	cache, mr := newProfileTestCache(t, inner, 0, 0)

	_, _ = cache.GetProfile(context.Background(), orgID)
	ttl := mr.TTL("profile:agency:" + orgID.String())
	assert.Greater(t, ttl, time.Duration(0))
}

// --- Edge cases for higher coverage ---

func TestProfileCache_CorruptCachedEntry_TreatedAsMiss(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	orgID := uuid.New()
	mr.Set("profile:agency:"+orgID.String(), "{broken")

	inner := newStubProfile(samplePresentProfile(orgID), nil)
	cache := adapter.NewCachedPublicProfileReader(client, inner, 60*time.Second, 30*time.Second)

	got, err := cache.GetProfile(context.Background(), orgID)
	require.NoError(t, err)
	assert.Equal(t, "Senior Backend Engineer", got.Title)
}

// --- Stampede protection ---

func TestProfileCache_Singleflight_CoalescesConcurrentMisses(t *testing.T) {
	// F.6 B10: stabilises the previously flaky variant. The original
	// test launched 100 goroutines back-to-back with `go func()` and
	// asserted exactly one inner call. Two real-world races defeated
	// that invariant under -race instrumentation:
	//
	//   1. Goroutine spawn scheduling can stretch over many ms when
	//      the OS scheduler is busy. A goroutine spawned late could
	//      reach tryGet AFTER the first inner call had finished AND
	//      released the singleflight slot — its subsequent group.Do
	//      starts a fresh inner call (legitimate behaviour, but
	//      defeats the coalescing test invariant).
	//   2. Inside one goroutine, a preemption between tryGet (miss)
	//      and group.Do can be longer than the inner-call delay,
	//      same outcome as case 1.
	//
	// Mitigation: synchronise all 100 goroutines on a starting gate
	// so they enter cache.GetProfile in a tight burst, AND lengthen
	// the inner-call window so even a worst-case scheduling delay
	// stays well inside it. The two together are sufficient to make
	// the test deterministic over 1000+ iterations under -race.
	orgID := uuid.New()
	inner := newStubProfile(samplePresentProfile(orgID), nil)
	// 250ms inner-call delay > any plausible scheduling jitter on a
	// busy CI box. The test still finishes in ~250ms total because
	// every goroutine after the first coalesces on the singleflight.
	inner.delay = 250 * time.Millisecond
	cache, _ := newProfileTestCache(t, inner, 60*time.Second, 30*time.Second)

	const n = 100

	// readyWG tracks "every goroutine has been spawned + reached the
	// gate". Closing `start` unblocks all of them at once so the
	// thundering-herd is genuinely concurrent rather than staggered
	// by goroutine-spawn time.
	var readyWG sync.WaitGroup
	readyWG.Add(n)
	start := make(chan struct{})

	var doneWG sync.WaitGroup
	doneWG.Add(n)
	var errs atomic.Int32

	for i := 0; i < n; i++ {
		go func() {
			defer doneWG.Done()
			readyWG.Done()
			<-start
			p, err := cache.GetProfile(context.Background(), orgID)
			if err != nil || p == nil {
				errs.Add(1)
			}
		}()
	}

	readyWG.Wait()
	close(start)
	doneWG.Wait()

	assert.Equal(t, int32(0), errs.Load())
	assert.Equal(t, 1, inner.callCount(orgID), "singleflight must coalesce a thundering herd into one DB call")
}
