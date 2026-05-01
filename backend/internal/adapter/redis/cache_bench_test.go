package redis_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"

	adapter "marketplace-backend/internal/adapter/redis"
	domainskill "marketplace-backend/internal/domain/skill"
)

// Benchmarks contrast the hot-path latency of a cache hit vs the
// cache miss path. The miss path includes the inner stub round-trip
// — production miss latency will be (Redis read + Postgres query +
// Redis write), so the absolute number is not directly comparable
// to live numbers, but the *ratio* is: a hit must be at least 5x
// faster than a miss for the cache to be worth its complexity.
//
// Run with:  go test -bench=BenchmarkCache -benchmem -run=^$ ./internal/adapter/redis

// --- Profile cache ---

func BenchmarkPublicProfileCache_Hit(b *testing.B) {
	mr, _ := miniredis.Run()
	defer mr.Close()
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	defer client.Close()

	orgID := uuid.New()
	inner := newStubProfile(samplePresentProfile(orgID), nil)
	cache := adapter.NewCachedPublicProfileReader(client, inner, 60*time.Second, 30*time.Second)

	// Prime the cache.
	_, _ = cache.GetProfile(context.Background(), orgID)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = cache.GetProfile(context.Background(), orgID)
	}
}

func BenchmarkPublicProfileCache_Miss(b *testing.B) {
	mr, _ := miniredis.Run()
	defer mr.Close()
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	defer client.Close()

	inner := newStubProfile(samplePresentProfile(uuid.New()), nil)
	cache := adapter.NewCachedPublicProfileReader(client, inner, 60*time.Second, 30*time.Second)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		// New uuid each iteration to force a miss every time.
		_, _ = cache.GetProfile(context.Background(), uuid.New())
	}
}

// --- Expertise cache ---

func BenchmarkExpertiseCache_Hit(b *testing.B) {
	mr, _ := miniredis.Run()
	defer mr.Close()
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	defer client.Close()

	orgID := uuid.New()
	inner := newStubExpertise([]string{"development", "design_ui_ux", "marketing_growth"}, nil)
	cache := adapter.NewCachedExpertiseReader(client, inner, 5*time.Minute)

	// Prime.
	_, _ = cache.ListByOrganization(context.Background(), orgID)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = cache.ListByOrganization(context.Background(), orgID)
	}
}

func BenchmarkExpertiseCache_Miss(b *testing.B) {
	mr, _ := miniredis.Run()
	defer mr.Close()
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	defer client.Close()

	inner := newStubExpertise([]string{"development", "design_ui_ux"}, nil)
	cache := adapter.NewCachedExpertiseReader(client, inner, 5*time.Minute)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = cache.ListByOrganization(context.Background(), uuid.New())
	}
}

// --- Skill catalog cache ---

func BenchmarkSkillCatalogCache_Hit(b *testing.B) {
	mr, _ := miniredis.Run()
	defer mr.Close()
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	defer client.Close()

	inner := newStubSkillCatalog()
	inner.listFn = func(_ string, _ int) ([]*domainskill.CatalogEntry, error) {
		// Simulate a 50-entry curated list.
		out := make([]*domainskill.CatalogEntry, 50)
		for i := range out {
			out[i] = &domainskill.CatalogEntry{SkillText: "react", DisplayText: "React", IsCurated: true}
		}
		return out, nil
	}
	cache := adapter.NewCachedSkillCatalogReader(client, inner, 10*time.Minute)

	// Prime.
	_, _ = cache.GetCuratedForExpertise(context.Background(), "development", 50)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = cache.GetCuratedForExpertise(context.Background(), "development", 50)
	}
}

func BenchmarkSkillCatalogCache_Miss(b *testing.B) {
	mr, _ := miniredis.Run()
	defer mr.Close()
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	defer client.Close()

	inner := newStubSkillCatalog()
	inner.listFn = func(_ string, _ int) ([]*domainskill.CatalogEntry, error) {
		out := make([]*domainskill.CatalogEntry, 50)
		for i := range out {
			out[i] = &domainskill.CatalogEntry{SkillText: "react"}
		}
		return out, nil
	}
	cache := adapter.NewCachedSkillCatalogReader(client, inner, 10*time.Minute)

	keys := []string{"development", "design_ui_ux", "marketing_growth", "data_ai_ml", "writing_translation"}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		// Cycle through expertise keys to force misses on a hot key
		// each time without colliding back into the cache.
		_, _ = cache.GetCuratedForExpertise(context.Background(), keys[i%len(keys)]+"_"+uuid.New().String(), 50)
	}
}
