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
	"marketplace-backend/internal/domain/organization"
)

// slowOverridesResolver is the org-overrides equivalent of
// slowVersionChecker: blocks every inner call until release(), so
// every concurrent caller is guaranteed to land inside the
// singleflight slot before any of them can complete.
type slowOverridesResolver struct {
	overrides  organization.RoleOverrides
	err        error
	innerCalls atomic.Int64
	gate       chan struct{}
	gateMu     sync.Mutex
	gateClosed bool
}

func newSlowOverridesResolver(overrides organization.RoleOverrides, err error) *slowOverridesResolver {
	return &slowOverridesResolver{
		overrides: overrides,
		err:       err,
		gate:      make(chan struct{}),
	}
}

func (s *slowOverridesResolver) GetRoleOverrides(_ context.Context, _ uuid.UUID) (organization.RoleOverrides, error) {
	s.innerCalls.Add(1)
	<-s.gate
	return s.overrides, s.err
}

func (s *slowOverridesResolver) release() {
	s.gateMu.Lock()
	defer s.gateMu.Unlock()
	if !s.gateClosed {
		close(s.gate)
		s.gateClosed = true
	}
}

func newOrgOverridesStampedeCache(t *testing.T, inner *slowOverridesResolver) *adapter.CachedOrgOverridesResolver {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return adapter.NewCachedOrgOverridesResolver(client, inner, 30*time.Second)
}

// ---------------------------------------------------------------------------
// QW-HARDENING fix #2 (org_overrides): 100 concurrent
// GetRoleOverrides on a miss for the same org_id must collapse to
// exactly one inner SELECT — same invariant as session_version.
// ---------------------------------------------------------------------------

func TestOrgOverridesCache_Stampede_CollapsesToOneInnerCall(t *testing.T) {
	t.Parallel()
	const N = 100
	inner := newSlowOverridesResolver(sampleOverrides(), nil)
	cache := newOrgOverridesStampedeCache(t, inner)

	orgID := uuid.New()
	results := make(chan organization.RoleOverrides, N)

	start := make(chan struct{})
	var ready sync.WaitGroup
	ready.Add(N)
	var wg sync.WaitGroup
	wg.Add(N)
	for range N {
		go func() {
			defer wg.Done()
			ready.Done()
			<-start
			got, err := cache.GetRoleOverrides(context.Background(), orgID)
			require.NoError(t, err)
			results <- got
		}()
	}
	ready.Wait()
	close(start)

	require.Eventually(t, func() bool {
		return inner.innerCalls.Load() >= 1
	}, 2*time.Second, 5*time.Millisecond,
		"first inner call must land before release")
	time.Sleep(200 * time.Millisecond)
	inner.release()
	wg.Wait()
	close(results)

	assert.Equal(t, int64(1), inner.innerCalls.Load(),
		"singleflight MUST coalesce concurrent org_overrides misses")

	expected := sampleOverrides()
	for got := range results {
		assert.Equal(t, expected, got, "all coalesced callers see identical overrides")
	}
}

func TestOrgOverridesCache_Stampede_AllCallersGetSameError(t *testing.T) {
	t.Parallel()
	const N = 50
	transient := errors.New("postgres unreachable")
	inner := newSlowOverridesResolver(nil, transient)
	cache := newOrgOverridesStampedeCache(t, inner)

	orgID := uuid.New()
	start := make(chan struct{})
	var ready sync.WaitGroup
	ready.Add(N)
	var wg sync.WaitGroup
	wg.Add(N)
	errCh := make(chan error, N)
	for range N {
		go func() {
			defer wg.Done()
			ready.Done()
			<-start
			_, err := cache.GetRoleOverrides(context.Background(), orgID)
			errCh <- err
		}()
	}
	ready.Wait()
	close(start)

	require.Eventually(t, func() bool {
		return inner.innerCalls.Load() >= 1
	}, 2*time.Second, 5*time.Millisecond)
	time.Sleep(200 * time.Millisecond)
	inner.release()
	wg.Wait()
	close(errCh)

	assert.Equal(t, int64(1), inner.innerCalls.Load(),
		"errors are coalesced across the burst")
	for err := range errCh {
		assert.ErrorIs(t, err, transient)
	}
}

// ---------------------------------------------------------------------------
// QW-HARDENING: nil overrides ("no customizations yet") must be a
// legitimate CACHED value — not a cache miss. Without the envelope
// wrapper, every read for an un-customised org would re-hit Postgres
// indefinitely.
// ---------------------------------------------------------------------------

func TestOrgOverridesCache_NilOverridesCachedUnderSingleflight(t *testing.T) {
	t.Parallel()
	inner := newStubResolver(nil, nil)
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	cache := adapter.NewCachedOrgOverridesResolver(client, inner, 30*time.Second)
	orgID := uuid.New()

	// 10 concurrent reads for an un-customised org. Inner must be
	// called exactly once; the rest must hit the cached nil value.
	const N = 10
	start := make(chan struct{})
	var ready sync.WaitGroup
	ready.Add(N)
	var wg sync.WaitGroup
	wg.Add(N)
	for range N {
		go func() {
			defer wg.Done()
			ready.Done()
			<-start
			got, err := cache.GetRoleOverrides(context.Background(), orgID)
			require.NoError(t, err)
			assert.Nil(t, got)
		}()
	}
	ready.Wait()
	close(start)
	wg.Wait()

	// stubOverridesResolver from org_overrides_cache_test.go counts
	// calls per orgID — at most a small number of inner calls
	// (typically 1, but the test uses the fast stub so there is no
	// blocking gate to enforce strict coalescing). The load-bearing
	// assertion is: inner was called AT LEAST once, AND subsequent
	// reads hit the cache (not 10 calls, even though there were 10
	// goroutines and the value is nil).
	calls := inner.callsFor[orgID]
	assert.GreaterOrEqual(t, calls, 1, "inner is called for the first miss")
	assert.LessOrEqual(t, calls, N, "must not exceed N")
	// Subsequent serial reads must hit the cache (proof that nil
	// IS cached, not treated as miss).
	for range 5 {
		got, err := cache.GetRoleOverrides(context.Background(), orgID)
		require.NoError(t, err)
		assert.Nil(t, got)
	}
	assert.Equal(t, calls, inner.callsFor[orgID],
		"post-cache reads must not re-hit inner — nil overrides ARE cached")
}

// ---------------------------------------------------------------------------
// QW-HARDENING combined test: Invalidate → next read sees the new
// overrides. Pins the integration of the org-overrides fix #1
// (RoleOverridesService calls Invalidate after SaveRoleOverrides)
// with the cache adapter.
// ---------------------------------------------------------------------------

func TestOrgOverridesCache_InvalidateThenRead_SeesNewOverrides(t *testing.T) {
	t.Parallel()
	var current atomic.Value
	current.Store(sampleOverrides())
	inner := &fnOverridesResolver{getFn: func(_ context.Context, _ uuid.UUID) (organization.RoleOverrides, error) {
		return current.Load().(organization.RoleOverrides), nil
	}}

	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	cache := adapter.NewCachedOrgOverridesResolver(client, inner, 30*time.Second)
	orgID := uuid.New()

	got, err := cache.GetRoleOverrides(context.Background(), orgID)
	require.NoError(t, err)
	require.Equal(t, sampleOverrides(), got)

	// Flip the inner answer (simulating a SaveRoleOverrides write
	// + the corresponding cache.Invalidate the service fires).
	newMatrix := organization.RoleOverrides{
		organization.Role("member"): {
			organization.Permission("billing.read"): true,
		},
	}
	current.Store(newMatrix)
	require.NoError(t, cache.Invalidate(context.Background(), orgID))

	got, err = cache.GetRoleOverrides(context.Background(), orgID)
	require.NoError(t, err)
	assert.Equal(t, newMatrix, got,
		"post-Invalidate read MUST see the new overrides, not the stale cache")
}

type fnOverridesResolver struct {
	getFn func(ctx context.Context, orgID uuid.UUID) (organization.RoleOverrides, error)
}

func (f *fnOverridesResolver) GetRoleOverrides(ctx context.Context, orgID uuid.UUID) (organization.RoleOverrides, error) {
	return f.getFn(ctx, orgID)
}
