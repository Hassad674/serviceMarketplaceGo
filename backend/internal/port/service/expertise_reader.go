package service

import (
	"context"

	"github.com/google/uuid"
)

// ExpertiseReader is the narrow read contract the public profile
// decoration paths depend on to fetch an organization's declared
// expertise list.
//
// Defined here so the Redis cache decorator
// (adapter/redis/expertise_cache.go) can satisfy the same interface
// as the app service — callers never know whether they are talking
// to the cache or the underlying service. Keeping the interface to
// the single ListByOrganization method matches the Interface
// Segregation principle: the search decoration path needs nothing
// else.
//
// Implementations MUST return a non-nil slice (empty when the org
// has not declared anything) so the JSON response carries "[]"
// rather than "null".
type ExpertiseReader interface {
	ListByOrganization(ctx context.Context, orgID uuid.UUID) ([]string, error)
}

// CacheInvalidatorByOrgID is the narrow write hook the app service
// fires after a successful expertise mutation. Cache adapters
// implement this so a cleared list is reflected on the very next
// read instead of waiting for the TTL to age out.
//
// Missing key MUST be treated as a no-op (return nil) — invalidating
// a never-cached org is a normal occurrence in the cold-start case.
type CacheInvalidatorByOrgID interface {
	Invalidate(ctx context.Context, orgID uuid.UUID) error
}
