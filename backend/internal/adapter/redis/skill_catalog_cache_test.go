package redis_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	adapter "marketplace-backend/internal/adapter/redis"
	domainskill "marketplace-backend/internal/domain/skill"
)

// stubSkillCatalogReader is a controllable inner.
type stubSkillCatalogReader struct {
	mu          sync.Mutex
	listFn      func(string, int) ([]*domainskill.CatalogEntry, error)
	countFn     func(string) (int, error)
	listCalls   map[string]int
	countCalls  map[string]int
	delay       time.Duration
}

func newStubSkillCatalog() *stubSkillCatalogReader {
	return &stubSkillCatalogReader{
		listCalls:  make(map[string]int),
		countCalls: make(map[string]int),
	}
}

func (s *stubSkillCatalogReader) GetCuratedForExpertise(_ context.Context, key string, limit int) ([]*domainskill.CatalogEntry, error) {
	s.mu.Lock()
	s.listCalls[key]++
	d := s.delay
	fn := s.listFn
	s.mu.Unlock()
	if d > 0 {
		time.Sleep(d)
	}
	if fn == nil {
		return nil, nil
	}
	return fn(key, limit)
}

func (s *stubSkillCatalogReader) CountCuratedForExpertise(_ context.Context, key string) (int, error) {
	s.mu.Lock()
	s.countCalls[key]++
	fn := s.countFn
	s.mu.Unlock()
	if fn == nil {
		return 0, nil
	}
	return fn(key)
}

func newSkillTestCache(t *testing.T, inner *stubSkillCatalogReader, ttl time.Duration) (*adapter.CachedSkillCatalogReader, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	return adapter.NewCachedSkillCatalogReader(client, inner, ttl), mr
}

// --- List: hit / miss / TTL ---

func TestSkillCatalog_List_MissThenHit(t *testing.T) {
	want := []*domainskill.CatalogEntry{
		{SkillText: "react", DisplayText: "React", IsCurated: true},
		{SkillText: "vue", DisplayText: "Vue", IsCurated: true},
	}
	inner := newStubSkillCatalog()
	inner.listFn = func(_ string, _ int) ([]*domainskill.CatalogEntry, error) { return want, nil }
	cache, _ := newSkillTestCache(t, inner, 10*time.Minute)

	first, err := cache.GetCuratedForExpertise(context.Background(), "development", 50)
	require.NoError(t, err)
	require.Len(t, first, 2)
	assert.Equal(t, "react", first[0].SkillText)

	second, err := cache.GetCuratedForExpertise(context.Background(), "development", 50)
	require.NoError(t, err)
	require.Len(t, second, 2)

	inner.mu.Lock()
	defer inner.mu.Unlock()
	assert.Equal(t, 1, inner.listCalls["development"])
}

func TestSkillCatalog_List_LimitsAreSeparateKeys(t *testing.T) {
	// limit=20 and limit=50 must not share a cache entry — different
	// payload sizes. The cache key includes the limit on purpose.
	inner := newStubSkillCatalog()
	inner.listFn = func(_ string, limit int) ([]*domainskill.CatalogEntry, error) {
		// Return `limit` placeholder entries so we can detect cross-talk.
		out := make([]*domainskill.CatalogEntry, limit)
		for i := range out {
			out[i] = &domainskill.CatalogEntry{SkillText: "x", IsCurated: true}
		}
		return out, nil
	}
	cache, _ := newSkillTestCache(t, inner, 10*time.Minute)

	got20, _ := cache.GetCuratedForExpertise(context.Background(), "development", 20)
	got50, _ := cache.GetCuratedForExpertise(context.Background(), "development", 50)

	assert.Len(t, got20, 20)
	assert.Len(t, got50, 50)
	inner.mu.Lock()
	defer inner.mu.Unlock()
	assert.Equal(t, 2, inner.listCalls["development"], "different limits must each hit the inner once")
}

func TestSkillCatalog_List_ExpiresAfterTTL(t *testing.T) {
	inner := newStubSkillCatalog()
	inner.listFn = func(_ string, _ int) ([]*domainskill.CatalogEntry, error) {
		return []*domainskill.CatalogEntry{{SkillText: "react"}}, nil
	}
	cache, mr := newSkillTestCache(t, inner, 10*time.Minute)

	_, _ = cache.GetCuratedForExpertise(context.Background(), "development", 50)
	mr.FastForward(11 * time.Minute)
	_, _ = cache.GetCuratedForExpertise(context.Background(), "development", 50)

	inner.mu.Lock()
	defer inner.mu.Unlock()
	assert.Equal(t, 2, inner.listCalls["development"])
}

func TestSkillCatalog_List_InnerError_NotCached(t *testing.T) {
	inner := newStubSkillCatalog()
	inner.listFn = func(_ string, _ int) ([]*domainskill.CatalogEntry, error) {
		return nil, errors.New("DB down")
	}
	cache, mr := newSkillTestCache(t, inner, 10*time.Minute)

	_, err := cache.GetCuratedForExpertise(context.Background(), "development", 50)
	require.Error(t, err)

	_, getErr := mr.Get("skills:curated:list:development:50")
	assert.Equal(t, miniredis.ErrKeyNotFound, getErr)
}

// --- Count: hit / miss ---

func TestSkillCatalog_Count_MissThenHit(t *testing.T) {
	inner := newStubSkillCatalog()
	inner.countFn = func(_ string) (int, error) { return 142, nil }
	cache, _ := newSkillTestCache(t, inner, 10*time.Minute)

	first, err := cache.CountCuratedForExpertise(context.Background(), "development")
	require.NoError(t, err)
	assert.Equal(t, 142, first)

	second, err := cache.CountCuratedForExpertise(context.Background(), "development")
	require.NoError(t, err)
	assert.Equal(t, 142, second)

	inner.mu.Lock()
	defer inner.mu.Unlock()
	assert.Equal(t, 1, inner.countCalls["development"])
}

func TestSkillCatalog_Count_InnerError_NotCached(t *testing.T) {
	inner := newStubSkillCatalog()
	inner.countFn = func(_ string) (int, error) { return 0, errors.New("DB down") }
	cache, mr := newSkillTestCache(t, inner, 10*time.Minute)

	_, err := cache.CountCuratedForExpertise(context.Background(), "development")
	require.Error(t, err)

	_, getErr := mr.Get("skills:curated:count:development")
	assert.Equal(t, miniredis.ErrKeyNotFound, getErr)
}

// --- Invalidation ---

func TestSkillCatalog_InvalidateExpertise_ClearsList_AndCount(t *testing.T) {
	inner := newStubSkillCatalog()
	inner.listFn = func(_ string, _ int) ([]*domainskill.CatalogEntry, error) {
		return []*domainskill.CatalogEntry{{SkillText: "react"}}, nil
	}
	inner.countFn = func(_ string) (int, error) { return 1, nil }
	cache, mr := newSkillTestCache(t, inner, 10*time.Minute)

	// Prime list (limit=20 + limit=50) and count.
	_, _ = cache.GetCuratedForExpertise(context.Background(), "development", 20)
	_, _ = cache.GetCuratedForExpertise(context.Background(), "development", 50)
	_, _ = cache.CountCuratedForExpertise(context.Background(), "development")

	// Sanity: entries exist.
	require.Contains(t, mr.Keys(), "skills:curated:list:development:20")
	require.Contains(t, mr.Keys(), "skills:curated:list:development:50")
	require.Contains(t, mr.Keys(), "skills:curated:count:development")

	require.NoError(t, cache.InvalidateExpertise(context.Background(), "development"))

	assert.NotContains(t, mr.Keys(), "skills:curated:list:development:20")
	assert.NotContains(t, mr.Keys(), "skills:curated:list:development:50")
	assert.NotContains(t, mr.Keys(), "skills:curated:count:development")
}

func TestSkillCatalog_InvalidateExpertise_DoesNotTouchOtherKeys(t *testing.T) {
	// Critical: invalidating "development" must NOT touch
	// "design_ui_ux" entries.
	inner := newStubSkillCatalog()
	inner.listFn = func(_ string, _ int) ([]*domainskill.CatalogEntry, error) {
		return []*domainskill.CatalogEntry{{SkillText: "react"}}, nil
	}
	cache, mr := newSkillTestCache(t, inner, 10*time.Minute)

	_, _ = cache.GetCuratedForExpertise(context.Background(), "development", 50)
	_, _ = cache.GetCuratedForExpertise(context.Background(), "design_ui_ux", 50)

	require.NoError(t, cache.InvalidateExpertise(context.Background(), "development"))

	assert.NotContains(t, mr.Keys(), "skills:curated:list:development:50")
	assert.Contains(t, mr.Keys(), "skills:curated:list:design_ui_ux:50",
		"unrelated expertise entries must not be flushed")
}

func TestSkillCatalog_InvalidateExpertise_MissingKeys_NoError(t *testing.T) {
	cache, _ := newSkillTestCache(t, newStubSkillCatalog(), 10*time.Minute)
	err := cache.InvalidateExpertise(context.Background(), "marketing_growth")
	assert.NoError(t, err)
}

func TestSkillCatalog_InvalidateAll_ClearsAllNamespaces(t *testing.T) {
	inner := newStubSkillCatalog()
	inner.listFn = func(_ string, _ int) ([]*domainskill.CatalogEntry, error) {
		return []*domainskill.CatalogEntry{{SkillText: "react"}}, nil
	}
	inner.countFn = func(_ string) (int, error) { return 1, nil }
	cache, mr := newSkillTestCache(t, inner, 10*time.Minute)

	_, _ = cache.GetCuratedForExpertise(context.Background(), "development", 50)
	_, _ = cache.GetCuratedForExpertise(context.Background(), "design_ui_ux", 50)
	_, _ = cache.CountCuratedForExpertise(context.Background(), "development")

	require.NoError(t, cache.InvalidateAll(context.Background()))

	for _, k := range mr.Keys() {
		assert.False(t,
			containsPrefix(k, "skills:curated:list:") || containsPrefix(k, "skills:curated:count:"),
			"InvalidateAll must wipe every skills:curated:* key, found: %s", k)
	}
}

func containsPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

// --- Resilience ---

func TestSkillCatalog_RedisDown_FallsThroughToInner(t *testing.T) {
	inner := newStubSkillCatalog()
	inner.listFn = func(_ string, _ int) ([]*domainskill.CatalogEntry, error) {
		return []*domainskill.CatalogEntry{{SkillText: "react"}}, nil
	}
	inner.countFn = func(_ string) (int, error) { return 1, nil }

	client := goredis.NewClient(&goredis.Options{
		Addr:        "127.0.0.1:1",
		DialTimeout: 150 * time.Millisecond,
	})
	t.Cleanup(func() { _ = client.Close() })

	cache := adapter.NewCachedSkillCatalogReader(client, inner, 10*time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	entries, err := cache.GetCuratedForExpertise(ctx, "development", 50)
	require.NoError(t, err)
	assert.Len(t, entries, 1)

	count, err := cache.CountCuratedForExpertise(ctx, "development")
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestSkillCatalog_ZeroTTL_FallsBackToDefault(t *testing.T) {
	inner := newStubSkillCatalog()
	inner.listFn = func(_ string, _ int) ([]*domainskill.CatalogEntry, error) {
		return []*domainskill.CatalogEntry{{SkillText: "react"}}, nil
	}
	cache, mr := newSkillTestCache(t, inner, 0)

	_, _ = cache.GetCuratedForExpertise(context.Background(), "development", 50)
	ttl := mr.TTL("skills:curated:list:development:50")
	assert.Greater(t, ttl, time.Duration(0))
}

// --- Edge cases for higher coverage ---

func TestSkillCatalog_CorruptList_TreatedAsMiss(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	mr.Set("skills:curated:list:development:50", "{broken")

	inner := newStubSkillCatalog()
	inner.listFn = func(_ string, _ int) ([]*domainskill.CatalogEntry, error) {
		return []*domainskill.CatalogEntry{{SkillText: "react"}}, nil
	}
	cache := adapter.NewCachedSkillCatalogReader(client, inner, 10*time.Minute)

	got, err := cache.GetCuratedForExpertise(context.Background(), "development", 50)
	require.NoError(t, err)
	assert.Len(t, got, 1)
}

func TestSkillCatalog_CorruptCount_TreatedAsMiss(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	mr.Set("skills:curated:count:development", "not-a-number")

	inner := newStubSkillCatalog()
	inner.countFn = func(_ string) (int, error) { return 42, nil }
	cache := adapter.NewCachedSkillCatalogReader(client, inner, 10*time.Minute)

	got, err := cache.CountCuratedForExpertise(context.Background(), "development")
	require.NoError(t, err)
	assert.Equal(t, 42, got)
}

// --- Stampede protection ---

func TestSkillCatalog_Singleflight_CoalescesConcurrentMisses(t *testing.T) {
	inner := newStubSkillCatalog()
	inner.delay = 50 * time.Millisecond
	inner.listFn = func(_ string, _ int) ([]*domainskill.CatalogEntry, error) {
		return []*domainskill.CatalogEntry{{SkillText: "react"}}, nil
	}
	cache, _ := newSkillTestCache(t, inner, 10*time.Minute)

	const n = 100
	var wg sync.WaitGroup
	var errs atomic.Int32
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			out, err := cache.GetCuratedForExpertise(context.Background(), "development", 50)
			if err != nil || len(out) == 0 {
				errs.Add(1)
			}
		}()
	}
	wg.Wait()

	assert.Equal(t, int32(0), errs.Load())
	inner.mu.Lock()
	defer inner.mu.Unlock()
	assert.Equal(t, 1, inner.listCalls["development"], "singleflight must coalesce a thundering herd into one DB call")
}
