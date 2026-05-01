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
	"marketplace-backend/internal/domain/freelanceprofile"
	"marketplace-backend/internal/domain/profile"
	"marketplace-backend/internal/port/repository"
)

// stubFreelanceProfileReader is a controllable inner reader.
type stubFreelanceProfileReader struct {
	mu    sync.Mutex
	view  *repository.FreelanceProfileView
	err   error
	calls map[uuid.UUID]int
	delay time.Duration
}

func newStubFreelance(view *repository.FreelanceProfileView, err error) *stubFreelanceProfileReader {
	return &stubFreelanceProfileReader{
		view:  view,
		err:   err,
		calls: make(map[uuid.UUID]int),
	}
}

func (s *stubFreelanceProfileReader) GetPublicByOrgID(_ context.Context, orgID uuid.UUID) (*repository.FreelanceProfileView, error) {
	s.mu.Lock()
	s.calls[orgID]++
	d := s.delay
	s.mu.Unlock()
	if d > 0 {
		time.Sleep(d)
	}
	return s.view, s.err
}

func (s *stubFreelanceProfileReader) callCount(orgID uuid.UUID) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls[orgID]
}

func newFreelanceTestCache(t *testing.T, inner *stubFreelanceProfileReader, ttl, negTTL time.Duration) (*adapter.CachedPublicFreelanceProfileReader, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	return adapter.NewCachedPublicFreelanceProfileReader(client, inner, ttl, negTTL), mr
}

func sampleFreelanceView(orgID uuid.UUID) *repository.FreelanceProfileView {
	return &repository.FreelanceProfileView{
		Profile: &freelanceprofile.Profile{
			ID:                 uuid.New(),
			OrganizationID:     orgID,
			Title:              "Senior Full-Stack",
			About:              "Hello world",
			AvailabilityStatus: profile.AvailabilityNow,
			ExpertiseDomains:   []string{"development", "design_ui_ux"},
		},
		Shared: repository.OrganizationSharedProfile{
			PhotoURL:              "https://cdn.example.com/me.png",
			City:                  "Paris",
			CountryCode:           "FR",
			LanguagesProfessional: []string{"en", "fr"},
		},
	}
}

// --- Hit / miss / TTL ---

func TestFreelanceCache_MissThenHit_OneInnerCall(t *testing.T) {
	orgID := uuid.New()
	inner := newStubFreelance(sampleFreelanceView(orgID), nil)
	cache, _ := newFreelanceTestCache(t, inner, 60*time.Second, 30*time.Second)

	first, err := cache.GetPublicByOrgID(context.Background(), orgID)
	require.NoError(t, err)
	assert.Equal(t, "Senior Full-Stack", first.Profile.Title)
	assert.Equal(t, "Paris", first.Shared.City)
	assert.Equal(t, []string{"development", "design_ui_ux"}, first.Profile.ExpertiseDomains)

	second, err := cache.GetPublicByOrgID(context.Background(), orgID)
	require.NoError(t, err)
	assert.Equal(t, "Senior Full-Stack", second.Profile.Title)

	assert.Equal(t, 1, inner.callCount(orgID), "second read must hit the cache")
}

func TestFreelanceCache_ExpiresAfterTTL(t *testing.T) {
	orgID := uuid.New()
	inner := newStubFreelance(sampleFreelanceView(orgID), nil)
	cache, mr := newFreelanceTestCache(t, inner, 60*time.Second, 30*time.Second)

	_, _ = cache.GetPublicByOrgID(context.Background(), orgID)
	mr.FastForward(61 * time.Second)
	_, _ = cache.GetPublicByOrgID(context.Background(), orgID)

	assert.Equal(t, 2, inner.callCount(orgID))
}

// --- Negative cache ---

func TestFreelanceCache_NotFound_IsNegativeCached(t *testing.T) {
	orgID := uuid.New()
	inner := newStubFreelance(nil, freelanceprofile.ErrProfileNotFound)
	cache, _ := newFreelanceTestCache(t, inner, 60*time.Second, 30*time.Second)

	for i := 0; i < 5; i++ {
		_, err := cache.GetPublicByOrgID(context.Background(), orgID)
		require.ErrorIs(t, err, freelanceprofile.ErrProfileNotFound)
	}
	assert.Equal(t, 1, inner.callCount(orgID), "negative cache must absorb 404 spam")
}

func TestFreelanceCache_NotFoundExpiresFaster(t *testing.T) {
	orgID := uuid.New()
	inner := newStubFreelance(nil, freelanceprofile.ErrProfileNotFound)
	cache, mr := newFreelanceTestCache(t, inner, 60*time.Second, 30*time.Second)

	_, err := cache.GetPublicByOrgID(context.Background(), orgID)
	require.ErrorIs(t, err, freelanceprofile.ErrProfileNotFound)

	mr.FastForward(31 * time.Second)
	_, err = cache.GetPublicByOrgID(context.Background(), orgID)
	require.ErrorIs(t, err, freelanceprofile.ErrProfileNotFound)
	assert.Equal(t, 2, inner.callCount(orgID), "negative entry must age out after negativeTTL")
}

func TestFreelanceCache_OtherInnerError_NotCached(t *testing.T) {
	orgID := uuid.New()
	inner := newStubFreelance(nil, errors.New("connection lost"))
	cache, mr := newFreelanceTestCache(t, inner, 60*time.Second, 30*time.Second)

	_, err := cache.GetPublicByOrgID(context.Background(), orgID)
	require.Error(t, err)
	require.NotErrorIs(t, err, freelanceprofile.ErrProfileNotFound)

	_, getErr := mr.Get("profile:freelance:" + orgID.String())
	assert.Equal(t, miniredis.ErrKeyNotFound, getErr)
}

// --- Invalidation ---

func TestFreelanceCache_Invalidate_ClearsHit(t *testing.T) {
	orgID := uuid.New()
	v1 := sampleFreelanceView(orgID)
	v1.Profile.Title = "First"
	inner := newStubFreelance(v1, nil)
	cache, mr := newFreelanceTestCache(t, inner, 60*time.Second, 30*time.Second)

	_, _ = cache.GetPublicByOrgID(context.Background(), orgID)

	v2 := sampleFreelanceView(orgID)
	v2.Profile.Title = "Updated"
	inner.mu.Lock()
	inner.view = v2
	inner.mu.Unlock()

	stale, _ := cache.GetPublicByOrgID(context.Background(), orgID)
	assert.Equal(t, "First", stale.Profile.Title)

	require.NoError(t, cache.Invalidate(context.Background(), orgID))

	_, getErr := mr.Get("profile:freelance:" + orgID.String())
	assert.Equal(t, miniredis.ErrKeyNotFound, getErr)

	fresh, _ := cache.GetPublicByOrgID(context.Background(), orgID)
	assert.Equal(t, "Updated", fresh.Profile.Title)
}

func TestFreelanceCache_Invalidate_ClearsNegative(t *testing.T) {
	orgID := uuid.New()
	inner := newStubFreelance(nil, freelanceprofile.ErrProfileNotFound)
	cache, _ := newFreelanceTestCache(t, inner, 60*time.Second, 30*time.Second)

	_, err := cache.GetPublicByOrgID(context.Background(), orgID)
	require.ErrorIs(t, err, freelanceprofile.ErrProfileNotFound)

	created := sampleFreelanceView(orgID)
	inner.mu.Lock()
	inner.view = created
	inner.err = nil
	inner.mu.Unlock()

	require.NoError(t, cache.Invalidate(context.Background(), orgID))

	got, err := cache.GetPublicByOrgID(context.Background(), orgID)
	require.NoError(t, err)
	assert.Equal(t, created.Profile.Title, got.Profile.Title)
}

func TestFreelanceCache_Invalidate_MissingKey_NoError(t *testing.T) {
	cache, _ := newFreelanceTestCache(t, newStubFreelance(nil, nil), 60*time.Second, 30*time.Second)
	err := cache.Invalidate(context.Background(), uuid.New())
	assert.NoError(t, err)
}

// --- Resilience ---

func TestFreelanceCache_RedisDown_FallsThroughToInner(t *testing.T) {
	orgID := uuid.New()
	inner := newStubFreelance(sampleFreelanceView(orgID), nil)
	client := goredis.NewClient(&goredis.Options{
		Addr:        "127.0.0.1:1",
		DialTimeout: 150 * time.Millisecond,
	})
	t.Cleanup(func() { _ = client.Close() })

	cache := adapter.NewCachedPublicFreelanceProfileReader(client, inner, 60*time.Second, 30*time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	got, err := cache.GetPublicByOrgID(ctx, orgID)
	require.NoError(t, err)
	require.NotNil(t, got)
}

func TestFreelanceCache_ZeroTTLs_FallBackToDefaults(t *testing.T) {
	orgID := uuid.New()
	inner := newStubFreelance(sampleFreelanceView(orgID), nil)
	cache, mr := newFreelanceTestCache(t, inner, 0, 0)

	_, _ = cache.GetPublicByOrgID(context.Background(), orgID)
	ttl := mr.TTL("profile:freelance:" + orgID.String())
	assert.Greater(t, ttl, time.Duration(0))
}

// --- Edge cases for higher coverage ---

func TestFreelanceCache_CorruptCachedEntry_TreatedAsMiss(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	orgID := uuid.New()
	mr.Set("profile:freelance:"+orgID.String(), "{broken")

	inner := newStubFreelance(sampleFreelanceView(orgID), nil)
	cache := adapter.NewCachedPublicFreelanceProfileReader(client, inner, 60*time.Second, 30*time.Second)

	got, err := cache.GetPublicByOrgID(context.Background(), orgID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "Senior Full-Stack", got.Profile.Title)
}

// --- Stampede protection ---

func TestFreelanceCache_Singleflight_Coalesces(t *testing.T) {
	orgID := uuid.New()
	inner := newStubFreelance(sampleFreelanceView(orgID), nil)
	inner.delay = 50 * time.Millisecond
	cache, _ := newFreelanceTestCache(t, inner, 60*time.Second, 30*time.Second)

	const n = 100
	var wg sync.WaitGroup
	var errs atomic.Int32
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			v, err := cache.GetPublicByOrgID(context.Background(), orgID)
			if err != nil || v == nil {
				errs.Add(1)
			}
		}()
	}
	wg.Wait()

	assert.Equal(t, int32(0), errs.Load())
	assert.Equal(t, 1, inner.callCount(orgID), "singleflight must coalesce a thundering herd into one DB call")
}
