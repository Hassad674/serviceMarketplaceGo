package redis

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"

	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/handler/middleware"
)

// DefaultSessionVersionCacheTTL is 30s — the upper bound on how long a
// stale session_version snapshot can survive after a session was
// explicitly bumped.
//
// PERF-AUDIT QW2: before this cache, every authenticated request paid
// one Postgres round-trip (SELECT session_version FROM users WHERE id = $1)
// to validate that the carried session/JWT had not been revoked. The
// query is correct but cheap-per-call only relative to network RTT —
// against Neon it adds ~10-15 ms of round-trip time + ~2-3 ms of planning
// on the hot path. With 4-5 such cache-able lookups stacking on every
// auth'd request, the per-page latency regression vs the pre-Phase B
// baseline was significant. Caching the result in Redis for 30s keeps
// the freshness contract intact (revocation propagates within at most
// 30s; the explicit Invalidate path collapses that to "next request").
const DefaultSessionVersionCacheTTL = 30 * time.Second

// sessionVersionKeyPrefix is the Redis namespace.
const sessionVersionKeyPrefix = "session_version:"

// CachedSessionVersionChecker wraps an inner middleware.SessionVersionChecker
// (the postgres adapter, in production) with a Redis-backed hot
// cache. Implements middleware.SessionVersionChecker so the auth
// middleware never learns that caching exists.
//
// Cache semantics — match CachedUserStateChecker so operators have one
// mental model for both:
//   - Hit: Redis value wins, no inner call.
//   - Miss: delegate, write the result back with TTL.
//   - Inner returns ErrUserNotFound: do NOT cache the negative result.
//     A user being recreated under the same id is not a scenario we
//     support, but caching "gone" would amplify any transient lookup
//     glitch into a 30s lockout.
//   - Redis read error: log + fall through to the inner reader so the
//     request keeps flowing.
//   - Redis write error: log, return the inner result.
//
// QW-HARDENING: a singleflight.Group coalesces concurrent misses on
// the same user id into a single inner call — under burst load (e.g.
// 100 concurrent authenticated requests for the same user landing
// just after a TTL expiry), a naive cache stamps 100 SELECTs onto
// Postgres. With singleflight, exactly one goroutine performs the
// inner call and every other waiter receives the same answer. See
// coalesceWithDoubleCheck for the inner-peek-after-miss recipe that
// closes the residual race when the winner finishes between the
// outer peek and the singleflight slot.
type CachedSessionVersionChecker struct {
	client *goredis.Client
	inner  middleware.SessionVersionChecker
	ttl    time.Duration
	group  singleflight.Group
}

// NewCachedSessionVersionChecker returns a Redis-fronted decorator
// over `inner`. Pass DefaultSessionVersionCacheTTL unless there is a
// strong reason to deviate.
func NewCachedSessionVersionChecker(
	client *goredis.Client,
	inner middleware.SessionVersionChecker,
	ttl time.Duration,
) *CachedSessionVersionChecker {
	if ttl <= 0 {
		ttl = DefaultSessionVersionCacheTTL
	}
	return &CachedSessionVersionChecker{
		client: client,
		inner:  inner,
		ttl:    ttl,
	}
}

// GetSessionVersion satisfies middleware.SessionVersionChecker.
//
// QW-HARDENING: delegates the outer-peek / singleflight / inner-peek /
// load orchestration to coalesceWithDoubleCheck so a stampede of
// concurrent misses for the same user id collapses to one inner call.
func (c *CachedSessionVersionChecker) GetSessionVersion(
	ctx context.Context,
	userID uuid.UUID,
) (int, error) {
	key := sessionVersionKeyPrefix + userID.String()
	return coalesceWithDoubleCheck(
		&c.group, key,
		func() (int, bool, error) {
			return c.peek(ctx, key, userID)
		},
		func() (int, error) {
			return c.load(ctx, key, userID)
		},
	)
}

// peek attempts a cache read. Returns:
//   - (version, true, nil)  → cache hit, return as-is.
//   - (0,       false, nil) → miss (or transient Redis blip, treated
//     as miss so the caller advances to load).
//
// The current cache design does not negative-cache ErrUserNotFound,
// so peek never returns (zero, true, err).
func (c *CachedSessionVersionChecker) peek(
	ctx context.Context,
	key string,
	userID uuid.UUID,
) (int, bool, error) {
	val, err := c.client.Get(ctx, key).Result()
	if err == nil {
		version, parseErr := strconv.Atoi(val)
		if parseErr == nil {
			return version, true, nil
		}
		// Corrupted entry — log and treat as miss; next write will
		// overwrite it.
		slog.Warn("session version cache: malformed payload, refreshing",
			"user_id", userID, "raw", val, "error", parseErr)
		return 0, false, nil
	}
	if !errors.Is(err, goredis.Nil) {
		slog.Warn("session version cache: redis get failed, falling back to inner",
			"user_id", userID, "error", err)
	}
	return 0, false, nil
}

// load consults the inner reader on a true miss and write-throughs
// the result on success. Errors are never cached (see semantics on
// the type doc).
func (c *CachedSessionVersionChecker) load(
	ctx context.Context,
	key string,
	userID uuid.UUID,
) (int, error) {
	version, err := c.inner.GetSessionVersion(ctx, userID)
	if err != nil {
		return version, err
	}
	if sErr := c.client.Set(ctx, key, strconv.Itoa(version), c.ttl).Err(); sErr != nil {
		slog.Warn("session version cache: redis set failed",
			"user_id", userID, "error", sErr)
	}
	return version, nil
}

// Invalidate evicts the cached entry for userID. Call this from any
// code path that bumps users.session_version (logout-all, role change,
// password rotation) so the next authenticated request sees the new
// version immediately instead of waiting for the TTL. Missing key is
// NOT an error.
func (c *CachedSessionVersionChecker) Invalidate(
	ctx context.Context,
	userID uuid.UUID,
) error {
	key := sessionVersionKeyPrefix + userID.String()
	if _, err := c.client.Del(ctx, key).Result(); err != nil {
		return fmt.Errorf("session version cache: invalidate: %w", err)
	}
	return nil
}

// Compile-time assertion: the cache implements the middleware contract.
var _ middleware.SessionVersionChecker = (*CachedSessionVersionChecker)(nil)

// Compile-time assertion: ErrUserNotFound is in the user package — the
// inner reader returns it when the row is gone. Kept here for
// discoverability when reading the file in isolation.
var _ = user.ErrUserNotFound
