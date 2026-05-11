package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"

	"marketplace-backend/internal/domain/organization"
	"marketplace-backend/internal/handler/middleware"
)

// DefaultOrgOverridesCacheTTL is 30s — the upper bound on how long a
// stale role_overrides JSONB snapshot can survive after the org row
// was edited.
//
// PERF-AUDIT QW1: before this cache, every authenticated request paid
// one Postgres round-trip (SELECT * FROM organizations WHERE id = $1)
// to compute the caller's effective permissions on every authenticated
// request. The original adapter (cmd/api/org_overrides_adapter.go)
// projects out a single JSONB column from the full row payload, but
// the wire cost is the same: a fresh round-trip + planner per request,
// adding ~10-15 ms on Neon. Caching the JSONB blob for 30s keeps the
// freshness contract aligned with the user_state cache while removing
// the per-request DB hit. An explicit Invalidate path is exposed so
// the role-permissions edit endpoint can collapse the propagation to
// "next request" instead of waiting for the TTL.
const DefaultOrgOverridesCacheTTL = 30 * time.Second

// orgOverridesKeyPrefix is the Redis namespace.
const orgOverridesKeyPrefix = "org_overrides:"

// CachedOrgOverridesResolver wraps an inner
// middleware.OrgOverridesResolver (the postgres-backed adapter, in
// production) with a Redis-backed hot cache. Implements
// middleware.OrgOverridesResolver so the auth middleware never learns
// that caching exists.
//
// Cache semantics — match CachedUserStateChecker so operators have one
// mental model:
//   - Hit: Redis value wins, no inner call.
//   - Miss: delegate, write the result back with TTL.
//   - Inner returns error: do NOT cache. A transient DB blip must not
//     poison the cache with "empty overrides" — the middleware's
//     fail-open policy already trusts the session snapshot when the
//     resolver errors out.
//   - Redis read error: log + fall through to the inner reader.
//   - Redis write error: log, return the inner result.
type CachedOrgOverridesResolver struct {
	client *goredis.Client
	inner  middleware.OrgOverridesResolver
	ttl    time.Duration
}

// NewCachedOrgOverridesResolver returns a Redis-fronted decorator over
// `inner`. Pass DefaultOrgOverridesCacheTTL unless there is a strong
// reason to deviate.
func NewCachedOrgOverridesResolver(
	client *goredis.Client,
	inner middleware.OrgOverridesResolver,
	ttl time.Duration,
) *CachedOrgOverridesResolver {
	if ttl <= 0 {
		ttl = DefaultOrgOverridesCacheTTL
	}
	return &CachedOrgOverridesResolver{
		client: client,
		inner:  inner,
		ttl:    ttl,
	}
}

// GetRoleOverrides satisfies middleware.OrgOverridesResolver.
func (c *CachedOrgOverridesResolver) GetRoleOverrides(
	ctx context.Context,
	orgID uuid.UUID,
) (organization.RoleOverrides, error) {
	key := orgOverridesKeyPrefix + orgID.String()

	// 1. Cache hit?
	val, err := c.client.Get(ctx, key).Result()
	if err == nil {
		var overrides organization.RoleOverrides
		if jErr := json.Unmarshal([]byte(val), &overrides); jErr == nil {
			return overrides, nil
		} else {
			slog.Warn("org overrides cache: malformed payload, refreshing",
				"org_id", orgID, "error", jErr)
		}
	} else if !errors.Is(err, goredis.Nil) {
		slog.Warn("org overrides cache: redis get failed, falling back to inner",
			"org_id", orgID, "error", err)
	}

	// 2. Cache miss — consult the inner resolver.
	overrides, ierr := c.inner.GetRoleOverrides(ctx, orgID)
	if ierr != nil {
		// Do not cache errors — the middleware fails open on resolver
		// errors and uses the session snapshot. Caching an empty value
		// here would convert that fail-open into a fail-closed
		// (no overrides → defaults only) for the entire TTL window.
		return overrides, ierr
	}

	// 3. Write-through. Best-effort.
	payload, mErr := json.Marshal(overrides)
	if mErr != nil {
		// Marshalling our own JSONB payload should never fail; log and
		// return the authoritative answer.
		slog.Warn("org overrides cache: marshal failed",
			"org_id", orgID, "error", mErr)
		return overrides, nil
	}
	if sErr := c.client.Set(ctx, key, payload, c.ttl).Err(); sErr != nil {
		slog.Warn("org overrides cache: redis set failed",
			"org_id", orgID, "error", sErr)
	}
	return overrides, nil
}

// Invalidate evicts the cached entry for orgID. Call this from any
// code path that mutates organizations.role_overrides (the
// role-permissions editor endpoint) so the next authenticated request
// sees the new overrides immediately. Missing key is NOT an error.
func (c *CachedOrgOverridesResolver) Invalidate(
	ctx context.Context,
	orgID uuid.UUID,
) error {
	key := orgOverridesKeyPrefix + orgID.String()
	if _, err := c.client.Del(ctx, key).Result(); err != nil {
		return fmt.Errorf("org overrides cache: invalidate: %w", err)
	}
	return nil
}

// Compile-time assertion: the cache implements the middleware contract.
var _ middleware.OrgOverridesResolver = (*CachedOrgOverridesResolver)(nil)
