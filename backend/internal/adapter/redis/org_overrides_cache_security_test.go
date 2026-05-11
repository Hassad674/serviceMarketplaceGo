package redis_test

// SEC-AUDIT-CACHE-PART2 — adversarial security tests for the QW-HARDENING
// post-merge state of org_overrides_cache.go. Mirror image of
// session_version_cache_security_test.go, adapted for the
// CachedOrgOverridesResolver type:
//
//   * Payload is JSON-marshalled organization.RoleOverrides (a map),
//     so the malformed-payload + race-vs-marshal surface is bigger.
//   * Negative cache semantics: any inner error short-circuits the
//     write — there is no distinct "not-found" sentinel; ANY non-nil
//     err must NOT poison the cache.
//
// Vectors covered:
//   A — Race between mutation+Invalidate and concurrent read
//   B — Invalidate fails but inner mutation succeeds
//   C — No poisoning after Redis recovery
//   D — Singleflight key isolation across org ids
//   E — Negative cache poisoning (transient inner error never cached)
//   G — Concurrent Invalidate idempotency

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"

	adapter "marketplace-backend/internal/adapter/redis"
	"marketplace-backend/internal/domain/organization"
)

// versionedOverridesResolver is a thread-safe inner that returns
// per-org overrides snapshots with optional latency and per-org
// inner-call accounting.
type versionedOverridesResolver struct {
	mu        sync.Mutex
	overrides map[uuid.UUID]organization.RoleOverrides
	calls     map[uuid.UUID]int
	delay     time.Duration
	err       map[uuid.UUID]error
}

func newVersionedOverridesResolver() *versionedOverridesResolver {
	return &versionedOverridesResolver{
		overrides: make(map[uuid.UUID]organization.RoleOverrides),
		calls:     make(map[uuid.UUID]int),
		err:       make(map[uuid.UUID]error),
	}
}

func (r *versionedOverridesResolver) GetRoleOverrides(_ context.Context, orgID uuid.UUID) (organization.RoleOverrides, error) {
	if r.delay > 0 {
		time.Sleep(r.delay)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls[orgID]++
	if e, ok := r.err[orgID]; ok && e != nil {
		return nil, e
	}
	return r.overrides[orgID], nil
}

func (r *versionedOverridesResolver) set(orgID uuid.UUID, o organization.RoleOverrides) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.overrides[orgID] = o
}

func (r *versionedOverridesResolver) setErr(orgID uuid.UUID, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.err[orgID] = err
}

func (r *versionedOverridesResolver) callsTo(orgID uuid.UUID) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.calls[orgID]
}

func newSecOrgOverridesCache(
	t *testing.T,
	inner *versionedOverridesResolver,
	ttl time.Duration,
) (*adapter.CachedOrgOverridesResolver, *miniredis.Miniredis, *goredis.Client) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	cache := adapter.NewCachedOrgOverridesResolver(client, inner, ttl)
	return cache, mr, client
}

// overridesV is a tiny helper that builds a non-empty RoleOverrides
// keyed by a single permission name embedding the version number, so
// the test can assert "this caller read the post-bump value" by
// looking at the map keys.
func overridesV(version int) organization.RoleOverrides {
	return organization.RoleOverrides{
		organization.Role("admin"): {
			organization.Permission(fmt.Sprintf("v%d", version)): true,
		},
	}
}

// ----------------------------------------------------------------------------
// Vector A — Race between mutation+Invalidate and concurrent read.
//
// Mirror of session_version vector A — assert that after Invalidate
// returns, every subsequent read reflects the new overrides snapshot.
// ----------------------------------------------------------------------------

func TestSecurityOrgOverrides_A_RaceMutationVsRead(t *testing.T) {
	t.Parallel()
	inner := newVersionedOverridesResolver()
	cache, _, _ := newSecOrgOverridesCache(t, inner, 30*time.Second)
	orgID := uuid.New()
	inner.set(orgID, overridesV(0))
	ctx := context.Background()

	// Warm.
	v, err := cache.GetRoleOverrides(ctx, orgID)
	require.NoError(t, err)
	require.Equal(t, overridesV(0), v)

	const iterations = 1000
	for i := 1; i <= iterations; i++ {
		var wg sync.WaitGroup
		wg.Add(1)
		readErrs := make(chan error, 1)
		go func() {
			defer wg.Done()
			_, e := cache.GetRoleOverrides(ctx, orgID)
			readErrs <- e
		}()

		// Mutation: swap snapshot + invalidate.
		inner.set(orgID, overridesV(i))
		invErr := cache.Invalidate(ctx, orgID)
		require.NoError(t, invErr, "iter %d: invalidate must not error", i)

		wg.Wait()
		select {
		case e := <-readErrs:
			require.NoError(t, e, "iter %d: concurrent read returned error", i)
		default:
		}

		// Post-invalidate read must reflect the bumped value.
		fresh, ferr := cache.GetRoleOverrides(ctx, orgID)
		require.NoError(t, ferr)
		require.Equalf(t, overridesV(i), fresh,
			"iter %d: stale read AFTER Invalidate", i)
	}
}

// ----------------------------------------------------------------------------
// Vector B — Invalidate fails but mutation succeeds.
//
// Close miniredis before calling Invalidate so the underlying DEL fails;
// the cache must surface a non-nil error rather than silently swallow
// the failure (which would hide stale-read leaks for up to 30s).
// ----------------------------------------------------------------------------

func TestSecurityOrgOverrides_B_InvalidateFailsSurfacesError(t *testing.T) {
	t.Parallel()
	inner := newVersionedOverridesResolver()
	cache, mr, _ := newSecOrgOverridesCache(t, inner, 30*time.Second)
	orgID := uuid.New()
	inner.set(orgID, overridesV(1))
	ctx := context.Background()

	v, err := cache.GetRoleOverrides(ctx, orgID)
	require.NoError(t, err)
	require.Equal(t, overridesV(1), v)

	mr.Close()
	inner.set(orgID, overridesV(2))

	invErr := cache.Invalidate(ctx, orgID)
	assert.Error(t, invErr,
		"vector B: Invalidate must surface Redis DEL failures so the call site can log/alert; "+
			"silently returning nil would hide a stampede of stale-read leaks for up to 30s")
}

// ----------------------------------------------------------------------------
// Vector C — No poisoning after Redis recovery.
// ----------------------------------------------------------------------------

func TestSecurityOrgOverrides_C_NoPoisoningAfterRedisRecovery(t *testing.T) {
	t.Parallel()
	inner := newVersionedOverridesResolver()
	cache, mr, _ := newSecOrgOverridesCache(t, inner, 30*time.Second)
	orgID := uuid.New()
	inner.set(orgID, overridesV(42))
	ctx := context.Background()

	mr.SetError("ECONNREFUSED simulated")

	v, err := cache.GetRoleOverrides(ctx, orgID)
	require.NoError(t, err)
	assert.Equal(t, overridesV(42), v,
		"vector C: under Redis blip, must fall through to inner")
	assert.GreaterOrEqual(t, inner.callsTo(orgID), 1)

	mr.SetError("")

	v2, err := cache.GetRoleOverrides(ctx, orgID)
	require.NoError(t, err)
	assert.Equal(t, overridesV(42), v2,
		"vector C: post-recovery read must equal DB value, never a stale poison")

	// Post-recovery write-through cached v42; mutating inner now must
	// NOT alter the served value.
	inner.set(orgID, overridesV(99))
	v3, err := cache.GetRoleOverrides(ctx, orgID)
	require.NoError(t, err)
	assert.Equal(t, overridesV(42), v3,
		"vector C: post-recovery write-through cached the correct value")
}

// ----------------------------------------------------------------------------
// Vector D — Singleflight key isolation across org ids.
//
// 100×2 concurrent misses across orgA and orgB. Each inner is called
// exactly once and no cross-key value leak.
// ----------------------------------------------------------------------------

func TestSecurityOrgOverrides_D_SingleflightKeyIsolation(t *testing.T) {
	t.Parallel()
	inner := newVersionedOverridesResolver()
	inner.delay = 50 * time.Millisecond
	cache, _, _ := newSecOrgOverridesCache(t, inner, 30*time.Second)
	orgA := uuid.New()
	orgB := uuid.New()
	inner.set(orgA, overridesV(111))
	inner.set(orgB, overridesV(222))
	ctx := context.Background()

	const callers = 100
	var gotA, gotB sync.Map
	var eg errgroup.Group
	start := make(chan struct{})

	for i := 0; i < callers; i++ {
		i := i
		eg.Go(func() error {
			<-start
			v, err := cache.GetRoleOverrides(ctx, orgA)
			if err != nil {
				return err
			}
			gotA.Store(i, v)
			return nil
		})
		eg.Go(func() error {
			<-start
			v, err := cache.GetRoleOverrides(ctx, orgB)
			if err != nil {
				return err
			}
			gotB.Store(i, v)
			return nil
		})
	}
	close(start)
	require.NoError(t, eg.Wait())

	assert.Equal(t, 1, inner.callsTo(orgA),
		"vector D: singleflight must collapse orgA's 100 concurrent misses to 1 inner call")
	assert.Equal(t, 1, inner.callsTo(orgB),
		"vector D: singleflight must collapse orgB's 100 concurrent misses to 1 inner call")

	gotA.Range(func(_, v interface{}) bool {
		assert.Equal(t, overridesV(111), v, "vector D: orgA caller got wrong value")
		return true
	})
	gotB.Range(func(_, v interface{}) bool {
		assert.Equal(t, overridesV(222), v, "vector D: orgB caller got wrong value")
		return true
	})
}

// ----------------------------------------------------------------------------
// Vector E — Negative cache poisoning.
//
// Inner returns a transient error. The cache must NOT write a key.
// 100 concurrent callers, all observe the error, no Redis key after.
// ----------------------------------------------------------------------------

func TestSecurityOrgOverrides_E_TransientErrorNotCached(t *testing.T) {
	t.Parallel()
	transient := errors.New("postgres unreachable")
	inner := newVersionedOverridesResolver()
	cache, mr, _ := newSecOrgOverridesCache(t, inner, 30*time.Second)
	orgID := uuid.New()
	inner.setErr(orgID, transient)
	ctx := context.Background()

	const callers = 100
	var eg errgroup.Group
	start := make(chan struct{})
	for i := 0; i < callers; i++ {
		eg.Go(func() error {
			<-start
			_, err := cache.GetRoleOverrides(ctx, orgID)
			if !errors.Is(err, transient) {
				return fmt.Errorf("expected transient error, got %v", err)
			}
			return nil
		})
	}
	close(start)
	require.NoError(t, eg.Wait())

	_, redisErr := mr.Get("org_overrides:" + orgID.String())
	assert.ErrorIs(t, redisErr, miniredis.ErrKeyNotFound,
		"vector E: transient inner errors MUST NOT poison the cache — Redis key must be absent")
	assert.GreaterOrEqual(t, inner.callsTo(orgID), 1,
		"vector E: inner must be called at least once")
}

// ----------------------------------------------------------------------------
// Vector G — Concurrent Invalidate idempotency.
// ----------------------------------------------------------------------------

func TestSecurityOrgOverrides_G_ConcurrentInvalidateIdempotent(t *testing.T) {
	t.Parallel()
	inner := newVersionedOverridesResolver()
	cache, mr, _ := newSecOrgOverridesCache(t, inner, 30*time.Second)
	orgID := uuid.New()
	inner.set(orgID, overridesV(7))
	ctx := context.Background()

	_, err := cache.GetRoleOverrides(ctx, orgID)
	require.NoError(t, err)

	const invalidators = 100
	var eg errgroup.Group
	var errCount atomic.Int64
	start := make(chan struct{})
	for i := 0; i < invalidators; i++ {
		eg.Go(func() error {
			<-start
			if e := cache.Invalidate(ctx, orgID); e != nil {
				errCount.Add(1)
				return e
			}
			return nil
		})
	}
	close(start)
	require.NoError(t, eg.Wait())
	assert.Equal(t, int64(0), errCount.Load(),
		"vector G: 100 concurrent Invalidate calls must all succeed")

	_, redisErr := mr.Get("org_overrides:" + orgID.String())
	assert.ErrorIs(t, redisErr, miniredis.ErrKeyNotFound,
		"vector G: after concurrent invalidate burst, the key must be gone")
}
