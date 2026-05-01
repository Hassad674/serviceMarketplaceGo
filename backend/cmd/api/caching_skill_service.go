package main

import (
	"context"

	"github.com/google/uuid"

	appskill "marketplace-backend/internal/app/skill"
	redisadapter "marketplace-backend/internal/adapter/redis"
	domainskill "marketplace-backend/internal/domain/skill"
)

// cachingSkillService is a tiny composite that routes the two
// highest-traffic, lowest-volatility catalog reads
// (GetCuratedForExpertise + CountCuratedForExpertise) through the
// Redis cache, while delegating every other method to the
// underlying skill app service.
//
// Why a hand-rolled composite instead of a generic decorator:
//   - The skill handler's local interface (skillService) demands
//     six methods; the cache only covers two. A generic decorator
//     would force every cache to implement the full surface.
//   - This file is the only place that needs the composition,
//     keeping the wiring obvious and the production interface
//     intact.
//
// The composite is stateless beyond the two pointers it holds, so
// a single instance is shared across all handler calls. It carries
// the same modularity story as the rest of cmd/api: cross-feature
// wiring lives in main.go (and helpers like this one), never in
// the feature packages themselves.
type cachingSkillService struct {
	inner *appskill.Service
	cache *redisadapter.CachedSkillCatalogReader
}

// newCachingSkillService wires the composite. Both arguments are
// required — there is no sane fallback for either.
func newCachingSkillService(inner *appskill.Service, cache *redisadapter.CachedSkillCatalogReader) *cachingSkillService {
	return &cachingSkillService{inner: inner, cache: cache}
}

// ---- Cached catalog reads ----

func (s *cachingSkillService) GetCuratedForExpertise(ctx context.Context, key string, limit int) ([]*domainskill.CatalogEntry, error) {
	return s.cache.GetCuratedForExpertise(ctx, key, limit)
}

func (s *cachingSkillService) CountCuratedForExpertise(ctx context.Context, key string) (int, error) {
	return s.cache.CountCuratedForExpertise(ctx, key)
}

// ---- Pass-through methods (cache is N/A or counter-productive) ----

func (s *cachingSkillService) Autocomplete(ctx context.Context, q string, limit int) ([]*domainskill.CatalogEntry, error) {
	// Autocomplete keys are too varied (every prefix the user might
	// type) for caching to be useful. The Postgres trigram index
	// already returns sub-10ms — bypass the cache.
	return s.inner.Autocomplete(ctx, q, limit)
}

func (s *cachingSkillService) GetProfileSkills(ctx context.Context, orgID uuid.UUID) ([]*domainskill.ProfileSkill, error) {
	return s.inner.GetProfileSkills(ctx, orgID)
}

func (s *cachingSkillService) ReplaceProfileSkills(ctx context.Context, in appskill.ReplaceProfileSkillsInput) error {
	return s.inner.ReplaceProfileSkills(ctx, in)
}

func (s *cachingSkillService) CreateUserSkill(ctx context.Context, in appskill.CreateUserSkillInput) (*domainskill.CatalogEntry, error) {
	return s.inner.CreateUserSkill(ctx, in)
}
