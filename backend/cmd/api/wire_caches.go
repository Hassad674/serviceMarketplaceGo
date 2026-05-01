package main

import (
	redisadapter "marketplace-backend/internal/adapter/redis"
	freelanceprofileapp "marketplace-backend/internal/app/freelanceprofile"
	profileapp "marketplace-backend/internal/app/profile"
	skillapp "marketplace-backend/internal/app/skill"

	goredis "github.com/redis/go-redis/v9"
)

// cachesWiring carries the public read caches put in front of the
// hottest profile / expertise / skill catalog endpoints. The
// underlying services are returned re-bound: each cache implements
// the Invalidate hook so the service can fire a cache delete after a
// successful DB write.
//
// CachingSkill is a tiny composite that routes the cached methods
// through Redis and delegates everything else to the underlying
// service — see caching_skill_service.go for the wrapper definition.
type cachesWiring struct {
	PublicProfileCache          *redisadapter.CachedPublicProfileReader
	PublicFreelanceProfileCache *redisadapter.CachedPublicFreelanceProfileReader
	ExpertiseCache              *redisadapter.CachedExpertiseReader
	ProfileSvc                  *profileapp.Service
	FreelanceProfileSvc         *freelanceprofileapp.Service
	ExpertiseSvc                *profileapp.ExpertiseService
	CachingSkill                *cachingSkillService
}

// cachesDeps captures the upstream services the cache layer wraps.
// Returned services in cachesWiring are the input services re-bound
// with the matching .WithCacheInvalidator setter — main.go must
// reassign its locals from the returned wiring.
type cachesDeps struct {
	Redis               *goredis.Client
	ProfileSvc          *profileapp.Service
	FreelanceProfileSvc *freelanceprofileapp.Service
	ExpertiseSvc        *profileapp.ExpertiseService
	SkillSvc            *skillapp.Service
}

// wireCaches brings up the Phase 4-M Redis cache-aside layer.
//
// ---- Phase 4-M: Redis cache-aside on hot read paths ----
//
// Each cache wraps the underlying app service via the decorator
// pattern (see adapter/redis/profile_cache.go for the rationale).
// Reads first consult Redis; misses fall through to the service
// and back-fill the entry. Writes go through the service directly
// — the service fires the cache's Invalidate hook AFTER a
// successful DB write (cache-aside contract — DB write succeeds
// → cache delete; reverse order opens a split-brain window).
//
// TTLs are tuned per signal volatility:
//   - profile:agency:{org}      60s (operator edits are rare)
//   - profile:freelance:{org}   60s (same)
//   - expertise:org:{org}       5min (lists change very rarely)
//   - skills:curated:{key}:{n}  10min (catalog is curator-seeded)
//
// Stampede protection: every cache uses a singleflight.Group so
// a thundering herd on a cold key triggers exactly one DB call.
//
// Negative caching: per-org profile caches absorb 404 spam by
// caching the not-found signal for 30s.
//
// Wired here (after wireSkillsAndPricing + wirePersonas) so the
// caches see the search-publisher-bound services produced by those
// helpers, then re-bind the affected handlers downstream.
func wireCaches(deps cachesDeps) cachesWiring {
	publicProfileCache := redisadapter.NewCachedPublicProfileReader(
		deps.Redis, deps.ProfileSvc,
		redisadapter.DefaultPublicProfileCacheTTL,
		redisadapter.DefaultPublicProfileNegativeTTL,
	)
	profileSvc := deps.ProfileSvc.WithCacheInvalidator(publicProfileCache)

	publicFreelanceProfileCache := redisadapter.NewCachedPublicFreelanceProfileReader(
		deps.Redis, deps.FreelanceProfileSvc,
		redisadapter.DefaultPublicProfileCacheTTL,
		redisadapter.DefaultPublicProfileNegativeTTL,
	)
	freelanceProfileSvc := deps.FreelanceProfileSvc.WithCacheInvalidator(publicFreelanceProfileCache)

	expertiseCache := redisadapter.NewCachedExpertiseReader(
		deps.Redis, deps.ExpertiseSvc, redisadapter.DefaultExpertiseCacheTTL,
	)
	expertiseSvc := deps.ExpertiseSvc.WithCacheInvalidator(expertiseCache)

	skillCatalogCache := redisadapter.NewCachedSkillCatalogReader(
		deps.Redis, deps.SkillSvc, redisadapter.DefaultSkillCatalogCacheTTL,
	)
	// The skill handler needs every method on the skill service —
	// the cache only covers the two highest-traffic catalog reads.
	// A tiny composite routes the cached methods through Redis and
	// delegates everything else to the underlying service. See
	// caching_skill_service.go for the wrapper definition.
	cachingSkillSvc := newCachingSkillService(deps.SkillSvc, skillCatalogCache)

	return cachesWiring{
		PublicProfileCache:          publicProfileCache,
		PublicFreelanceProfileCache: publicFreelanceProfileCache,
		ExpertiseCache:              expertiseCache,
		ProfileSvc:                  profileSvc,
		FreelanceProfileSvc:         freelanceProfileSvc,
		ExpertiseSvc:                expertiseSvc,
		CachingSkill:                cachingSkillSvc,
	}
}
