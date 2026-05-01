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
)

// stubExpertiseReader is a controllable ExpertiseReader used to
// assert the cache's delegation behaviour.
type stubExpertiseReader struct {
	mu       sync.Mutex
	keys     []string
	err      error
	calls    map[uuid.UUID]int
	delay    time.Duration // simulates a slow DB query for stampede tests
	hitDelay func()        // optional hook (alternative to fixed sleep)
}

func newStubExpertise(keys []string, err error) *stubExpertiseReader {
	return &stubExpertiseReader{
		keys:  keys,
		err:   err,
		calls: make(map[uuid.UUID]int),
	}
}

func (s *stubExpertiseReader) ListByOrganization(_ context.Context, orgID uuid.UUID) ([]string, error) {
	s.mu.Lock()
	s.calls[orgID]++
	d := s.delay
	hook := s.hitDelay
	s.mu.Unlock()
	if d > 0 {
		time.Sleep(d)
	}
	if hook != nil {
		hook()
	}
	return s.keys, s.err
}

func (s *stubExpertiseReader) callCount(orgID uuid.UUID) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls[orgID]
}

func newExpertiseTestCache(t *testing.T, inner *stubExpertiseReader, ttl time.Duration) (*adapter.CachedExpertiseReader, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	return adapter.NewCachedExpertiseReader(client, inner, ttl), mr
}

// --- Hit / miss / TTL / invalidation ---

func TestExpertiseCache_MissThenHit_OneInnerCall(t *testing.T) {
	inner := newStubExpertise([]string{"development", "design_ui_ux"}, nil)
	cache, _ := newExpertiseTestCache(t, inner, 5*time.Minute)
	orgID := uuid.New()

	first, err := cache.ListByOrganization(context.Background(), orgID)
	require.NoError(t, err)
	assert.Equal(t, []string{"development", "design_ui_ux"}, first)

	second, err := cache.ListByOrganization(context.Background(), orgID)
	require.NoError(t, err)
	assert.Equal(t, []string{"development", "design_ui_ux"}, second)

	assert.Equal(t, 1, inner.callCount(orgID), "second call must come from cache")
}

func TestExpertiseCache_EmptyListIsCached(t *testing.T) {
	// An org with no declared expertise still spends one DB call per
	// page view if we don't cache the empty answer. Ensure we DO cache
	// the empty case — it's the most common one for fresh accounts.
	inner := newStubExpertise([]string{}, nil)
	cache, _ := newExpertiseTestCache(t, inner, 5*time.Minute)
	orgID := uuid.New()

	for i := 0; i < 3; i++ {
		got, err := cache.ListByOrganization(context.Background(), orgID)
		require.NoError(t, err)
		assert.Equal(t, []string{}, got)
	}
	assert.Equal(t, 1, inner.callCount(orgID))
}

func TestExpertiseCache_NilFromInner_CachedAsEmpty(t *testing.T) {
	// The repository contract says non-nil empty, but defensive: a nil
	// slice from inner must surface as an empty (non-nil) slice for
	// stable JSON output and must still cache.
	inner := newStubExpertise(nil, nil)
	cache, _ := newExpertiseTestCache(t, inner, 5*time.Minute)
	orgID := uuid.New()

	got, err := cache.ListByOrganization(context.Background(), orgID)
	require.NoError(t, err)
	assert.NotNil(t, got)
	assert.Empty(t, got)

	got2, _ := cache.ListByOrganization(context.Background(), orgID)
	assert.Equal(t, got, got2)
	assert.Equal(t, 1, inner.callCount(orgID))
}

func TestExpertiseCache_ExpiresAfterTTL(t *testing.T) {
	inner := newStubExpertise([]string{"development"}, nil)
	cache, mr := newExpertiseTestCache(t, inner, 5*time.Minute)
	orgID := uuid.New()

	_, _ = cache.ListByOrganization(context.Background(), orgID)
	mr.FastForward(6 * time.Minute)
	_, _ = cache.ListByOrganization(context.Background(), orgID)

	assert.Equal(t, 2, inner.callCount(orgID), "entry must expire after TTL")
}

func TestExpertiseCache_InnerError_NotCached(t *testing.T) {
	inner := newStubExpertise(nil, errors.New("db lost"))
	cache, mr := newExpertiseTestCache(t, inner, 5*time.Minute)
	orgID := uuid.New()

	_, err := cache.ListByOrganization(context.Background(), orgID)
	require.Error(t, err)

	// Error path must NOT pin the cache — a transient DB blip should
	// not leave the org without expertise for the full TTL.
	_, getErr := mr.Get("expertise:org:" + orgID.String())
	assert.Equal(t, miniredis.ErrKeyNotFound, getErr)
}

func TestExpertiseCache_Invalidate_ClearsEntry(t *testing.T) {
	inner := newStubExpertise([]string{"development"}, nil)
	cache, mr := newExpertiseTestCache(t, inner, 5*time.Minute)
	orgID := uuid.New()

	// Prime cache.
	_, _ = cache.ListByOrganization(context.Background(), orgID)

	// Flip the inner answer; cache still serves the stale value.
	inner.mu.Lock()
	inner.keys = []string{"design_ui_ux"}
	inner.mu.Unlock()

	stale, _ := cache.ListByOrganization(context.Background(), orgID)
	assert.Equal(t, []string{"development"}, stale, "stale hit expected before invalidation")

	require.NoError(t, cache.Invalidate(context.Background(), orgID))

	_, getErr := mr.Get("expertise:org:" + orgID.String())
	assert.Equal(t, miniredis.ErrKeyNotFound, getErr, "invalidated entry must be gone")

	fresh, _ := cache.ListByOrganization(context.Background(), orgID)
	assert.Equal(t, []string{"design_ui_ux"}, fresh, "fresh read must reflect new state")
}

func TestExpertiseCache_Invalidate_MissingKey_NoError(t *testing.T) {
	inner := newStubExpertise(nil, nil)
	cache, _ := newExpertiseTestCache(t, inner, 5*time.Minute)

	err := cache.Invalidate(context.Background(), uuid.New())
	assert.NoError(t, err, "invalidating an absent key must be a no-op")
}

func TestExpertiseCache_RedisDown_FallsThroughToInner(t *testing.T) {
	// Redis pointed at a closed port — the cache MUST keep serving
	// reads via the inner so a Redis outage never takes down profile
	// reads.
	inner := newStubExpertise([]string{"development"}, nil)
	client := goredis.NewClient(&goredis.Options{
		Addr:        "127.0.0.1:1",
		DialTimeout: 150 * time.Millisecond,
	})
	t.Cleanup(func() { _ = client.Close() })

	cache := adapter.NewCachedExpertiseReader(client, inner, 5*time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	keys, err := cache.ListByOrganization(ctx, uuid.New())
	require.NoError(t, err, "Redis down must not fail the request")
	assert.Equal(t, []string{"development"}, keys)
}

func TestExpertiseCache_ZeroTTLFallsBackToDefault(t *testing.T) {
	inner := newStubExpertise([]string{"development"}, nil)
	cache, mr := newExpertiseTestCache(t, inner, 0)

	orgID := uuid.New()
	_, _ = cache.ListByOrganization(context.Background(), orgID)
	ttl := mr.TTL("expertise:org:" + orgID.String())
	assert.Greater(t, ttl, time.Duration(0))
}

// --- Edge cases for higher coverage ---

func TestExpertiseCache_CorruptCachedEntry_TreatedAsMiss(t *testing.T) {
	// A corrupt JSON blob (e.g. a bug in an older release) must NOT
	// crash the read path; the cache should fall through to the
	// inner reader and overwrite with a fresh entry.
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	orgID := uuid.New()
	mr.Set("expertise:org:"+orgID.String(), "{not valid json")

	inner := newStubExpertise([]string{"development"}, nil)
	cache := adapter.NewCachedExpertiseReader(client, inner, 5*time.Minute)

	got, err := cache.ListByOrganization(context.Background(), orgID)
	require.NoError(t, err)
	assert.Equal(t, []string{"development"}, got)
	// Subsequent read must hit the freshly-written entry.
	got2, _ := cache.ListByOrganization(context.Background(), orgID)
	assert.Equal(t, got, got2)
	assert.Equal(t, 1, inner.callCount(orgID), "second read must hit the rewritten cache, not the inner")
}

// --- Stampede protection ---

func TestExpertiseCache_Singleflight_CoalescesConcurrentMisses(t *testing.T) {
	// 50 concurrent readers on a cold key must produce exactly 1
	// inner call thanks to singleflight.
	inner := newStubExpertise([]string{"development"}, nil)
	inner.delay = 50 * time.Millisecond // long enough to overlap callers
	cache, _ := newExpertiseTestCache(t, inner, 5*time.Minute)
	orgID := uuid.New()

	const n = 50
	var wg sync.WaitGroup
	var errs atomic.Int32
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			keys, err := cache.ListByOrganization(context.Background(), orgID)
			if err != nil || len(keys) != 1 {
				errs.Add(1)
			}
		}()
	}
	wg.Wait()

	assert.Equal(t, int32(0), errs.Load(), "all readers must succeed")
	assert.Equal(t, 1, inner.callCount(orgID), "singleflight must coalesce concurrent misses into one inner call")
}
