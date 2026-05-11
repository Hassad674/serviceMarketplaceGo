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
	"marketplace-backend/internal/domain/organization"
)

// stubOverridesResolver is a controllable
// middleware.OrgOverridesResolver used to assert the cache's
// delegation behaviour and count inner calls per org id.
type stubOverridesResolver struct {
	overrides organization.RoleOverrides
	err       error
	callsFor  map[uuid.UUID]int
}

func newStubResolver(overrides organization.RoleOverrides, err error) *stubOverridesResolver {
	return &stubOverridesResolver{
		overrides: overrides,
		err:       err,
		callsFor:  make(map[uuid.UUID]int),
	}
}

func (s *stubOverridesResolver) GetRoleOverrides(_ context.Context, orgID uuid.UUID) (organization.RoleOverrides, error) {
	s.callsFor[orgID]++
	return s.overrides, s.err
}

func newTestOrgOverridesCache(
	t *testing.T,
	inner *stubOverridesResolver,
	ttl time.Duration,
) (*adapter.CachedOrgOverridesResolver, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	cache := adapter.NewCachedOrgOverridesResolver(client, inner, ttl)
	return cache, mr
}

func sampleOverrides() organization.RoleOverrides {
	// Minimal non-empty payload — keep the shape opaque so the test
	// only verifies round-trip behaviour, not the schema.
	return organization.RoleOverrides{
		organization.Role("admin"): {
			organization.Permission("team.manage"): true,
			organization.Permission("billing.read"): true,
		},
	}
}

func TestOrgOverridesCache_MissThenHit(t *testing.T) {
	t.Parallel()
	inner := newStubResolver(sampleOverrides(), nil)
	cache, _ := newTestOrgOverridesCache(t, inner, 30*time.Second)

	orgID := uuid.New()
	ctx := context.Background()

	// Miss → delegate → cache.
	got, err := cache.GetRoleOverrides(ctx, orgID)
	require.NoError(t, err)
	assert.Equal(t, sampleOverrides(), got)
	assert.Equal(t, 1, inner.callsFor[orgID])

	// Hit → no inner call.
	got, err = cache.GetRoleOverrides(ctx, orgID)
	require.NoError(t, err)
	assert.Equal(t, sampleOverrides(), got)
	assert.Equal(t, 1, inner.callsFor[orgID])
}

func TestOrgOverridesCache_TTLExpires(t *testing.T) {
	t.Parallel()
	inner := newStubResolver(sampleOverrides(), nil)
	cache, mr := newTestOrgOverridesCache(t, inner, 5*time.Second)

	orgID := uuid.New()
	ctx := context.Background()

	_, err := cache.GetRoleOverrides(ctx, orgID)
	require.NoError(t, err)

	mr.FastForward(6 * time.Second)

	_, err = cache.GetRoleOverrides(ctx, orgID)
	require.NoError(t, err)
	assert.Equal(t, 2, inner.callsFor[orgID])
}

func TestOrgOverridesCache_DefaultTTLAppliedWhenZero(t *testing.T) {
	t.Parallel()
	inner := newStubResolver(sampleOverrides(), nil)
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	cache := adapter.NewCachedOrgOverridesResolver(client, inner, 0)
	orgID := uuid.New()
	ctx := context.Background()

	_, err = cache.GetRoleOverrides(ctx, orgID)
	require.NoError(t, err)

	ttl := mr.TTL("org_overrides:" + orgID.String())
	assert.InDelta(t, adapter.DefaultOrgOverridesCacheTTL.Seconds(), ttl.Seconds(), 1.0)
}

func TestOrgOverridesCache_DoesNotCacheError(t *testing.T) {
	t.Parallel()
	transient := errors.New("postgres unreachable")
	inner := newStubResolver(nil, transient)
	cache, mr := newTestOrgOverridesCache(t, inner, 30*time.Second)
	orgID := uuid.New()
	ctx := context.Background()

	_, err := cache.GetRoleOverrides(ctx, orgID)
	assert.ErrorIs(t, err, transient)

	_, redisErr := mr.Get("org_overrides:" + orgID.String())
	assert.ErrorIs(t, redisErr, miniredis.ErrKeyNotFound, "errors must NEVER be cached")
}

func TestOrgOverridesCache_NilOverridesCached(t *testing.T) {
	// A brand-new org with no overrides (nil map) must still be cached
	// so subsequent reads don't re-hit Postgres.
	t.Parallel()
	inner := newStubResolver(nil, nil)
	cache, _ := newTestOrgOverridesCache(t, inner, 30*time.Second)
	orgID := uuid.New()
	ctx := context.Background()

	got, err := cache.GetRoleOverrides(ctx, orgID)
	require.NoError(t, err)
	assert.Nil(t, got)

	got, err = cache.GetRoleOverrides(ctx, orgID)
	require.NoError(t, err)
	assert.Nil(t, got)
	assert.Equal(t, 1, inner.callsFor[orgID], "inner called once; nil overrides ARE cached")
}

func TestOrgOverridesCache_MalformedPayloadRefreshes(t *testing.T) {
	t.Parallel()
	inner := newStubResolver(sampleOverrides(), nil)
	cache, mr := newTestOrgOverridesCache(t, inner, 30*time.Second)
	orgID := uuid.New()

	require.NoError(t, mr.Set("org_overrides:"+orgID.String(), "{not json"))

	got, err := cache.GetRoleOverrides(context.Background(), orgID)
	require.NoError(t, err)
	assert.Equal(t, sampleOverrides(), got)
	assert.Equal(t, 1, inner.callsFor[orgID])
}

func TestOrgOverridesCache_Invalidate(t *testing.T) {
	t.Parallel()
	inner := newStubResolver(sampleOverrides(), nil)
	cache, mr := newTestOrgOverridesCache(t, inner, 30*time.Second)
	orgID := uuid.New()
	ctx := context.Background()

	_, err := cache.GetRoleOverrides(ctx, orgID)
	require.NoError(t, err)

	require.NoError(t, cache.Invalidate(ctx, orgID))
	_, err = mr.Get("org_overrides:" + orgID.String())
	assert.ErrorIs(t, err, miniredis.ErrKeyNotFound)

	// Missing key — no error.
	require.NoError(t, cache.Invalidate(ctx, uuid.New()))
}
