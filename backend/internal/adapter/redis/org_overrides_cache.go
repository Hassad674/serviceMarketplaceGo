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
	"golang.org/x/sync/singleflight"

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

// orgOverridesEnvelope wraps the cached payload so the helper's
// generic peek/load shape can distinguish "cache miss" from "cache
// hit returning nil overrides" — both look the same when the type is
// `organization.RoleOverrides` (a map) directly. The envelope is a
// pointer; coalesceWithDoubleCheck's nil-check on the result then
// only fires when load itself returned a nil envelope.
type orgOverridesEnvelope struct {
	Overrides organization.RoleOverrides
}

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
//
// QW-HARDENING: a singleflight.Group coalesces concurrent misses on
// the same org id into a single inner call — under burst load (e.g.
// the 30s TTL expiring while 100 authenticated requests for the same
// org are in flight), a naive cache stamps 100 SELECTs onto Postgres.
// With singleflight, exactly one goroutine performs the inner call
// and every other waiter receives the same answer. See
// coalesceWithDoubleCheck for the inner-peek-after-miss recipe that
// closes the residual race when the winner finishes between the
// outer peek and the singleflight slot.
type CachedOrgOverridesResolver struct {
	client *goredis.Client
	inner  middleware.OrgOverridesResolver
	ttl    time.Duration
	group  singleflight.Group
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
//
// QW-HARDENING: delegates the outer-peek / singleflight / inner-peek /
// load orchestration to coalesceWithDoubleCheck. The wrapping
// envelope lets the helper carry a nil map ("no overrides yet") as a
// legitimate cached value without colliding with the miss sentinel.
func (c *CachedOrgOverridesResolver) GetRoleOverrides(
	ctx context.Context,
	orgID uuid.UUID,
) (organization.RoleOverrides, error) {
	key := orgOverridesKeyPrefix + orgID.String()
	env, err := coalesceWithDoubleCheck(
		&c.group, key,
		func() (*orgOverridesEnvelope, bool, error) {
			return c.peek(ctx, key, orgID)
		},
		func() (*orgOverridesEnvelope, error) {
			return c.load(ctx, key, orgID)
		},
	)
	if err != nil {
		return nil, err
	}
	if env == nil {
		// coalesceWithDoubleCheck normalises (nil, nil) to the typed
		// zero; for *orgOverridesEnvelope that means "load returned
		// nil envelope, nil error" — surface nil overrides.
		return nil, nil
	}
	return env.Overrides, nil
}

// peek attempts a cache read. Returns:
//   - (envelope, true, nil) → cache hit; envelope.Overrides may be nil.
//   - (nil,      false, nil) → miss (or transient Redis blip).
func (c *CachedOrgOverridesResolver) peek(
	ctx context.Context,
	key string,
	orgID uuid.UUID,
) (*orgOverridesEnvelope, bool, error) {
	val, err := c.client.Get(ctx, key).Result()
	if err == nil {
		var overrides organization.RoleOverrides
		if jErr := json.Unmarshal([]byte(val), &overrides); jErr == nil {
			return &orgOverridesEnvelope{Overrides: overrides}, true, nil
		} else {
			slog.Warn("org overrides cache: malformed payload, refreshing",
				"org_id", orgID, "error", jErr)
			return nil, false, nil
		}
	}
	if !errors.Is(err, goredis.Nil) {
		slog.Warn("org overrides cache: redis get failed, falling back to inner",
			"org_id", orgID, "error", err)
	}
	return nil, false, nil
}

// load consults the inner resolver on a true miss and write-throughs
// the result on success. Errors are never cached (see semantics on
// the type doc).
func (c *CachedOrgOverridesResolver) load(
	ctx context.Context,
	key string,
	orgID uuid.UUID,
) (*orgOverridesEnvelope, error) {
	overrides, err := c.inner.GetRoleOverrides(ctx, orgID)
	if err != nil {
		return nil, err
	}
	payload, mErr := json.Marshal(overrides)
	if mErr != nil {
		// Marshalling our own JSONB payload should never fail; log and
		// return the authoritative answer.
		slog.Warn("org overrides cache: marshal failed",
			"org_id", orgID, "error", mErr)
		return &orgOverridesEnvelope{Overrides: overrides}, nil
	}
	if sErr := c.client.Set(ctx, key, payload, c.ttl).Err(); sErr != nil {
		slog.Warn("org overrides cache: redis set failed",
			"org_id", orgID, "error", sErr)
	}
	return &orgOverridesEnvelope{Overrides: overrides}, nil
}

// Invalidate evicts the cached entry for orgID. Call this from any
// code path that mutates organizations.role_overrides (the
// role-permissions editor endpoint) so the next authenticated request
// sees the new overrides immediately. Missing key is NOT an error.
//
// QW-HARDENING: also `Forget`s the singleflight slot. Without this,
// a goroutine that joined an in-flight inner call BEFORE the
// invalidate would receive the pre-save overrides. Forget detaches
// the slot so the next coalesced burst re-reads the inner — closing
// the residual race between in-flight reads and the concurrent save.
func (c *CachedOrgOverridesResolver) Invalidate(
	ctx context.Context,
	orgID uuid.UUID,
) error {
	key := orgOverridesKeyPrefix + orgID.String()
	c.group.Forget(key)
	if _, err := c.client.Del(ctx, key).Result(); err != nil {
		return fmt.Errorf("org overrides cache: invalidate: %w", err)
	}
	return nil
}

// Compile-time assertion: the cache implements the middleware contract.
var _ middleware.OrgOverridesResolver = (*CachedOrgOverridesResolver)(nil)
