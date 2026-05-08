package redis

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"

	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/handler/middleware"
)

// DefaultUserStateCacheTTL is 30s — the upper bound on how long a
// stale (is_admin, status) snapshot can survive after a DB change.
//
// Tuning rationale:
//   - 30s is short enough that operator tooling promoting / demoting
//     a user does not require a coordinated cache flush; the worst
//     case is a 30s wait before the new state takes effect.
//   - 30s is long enough that a busy admin (10 req/s) absorbs the
//     fan-out via cache hits — the expected DB cost stays at one
//     primary-key lookup per user every 30s, regardless of traffic.
//   - The fix can also be accelerated by an explicit invalidation
//     (Invalidate) the moment a code path mutates is_admin/status,
//     so the TTL is purely a safety net for direct-SQL operator
//     edits that bypass the application layer.
const DefaultUserStateCacheTTL = 30 * time.Second

// userStateKeyPrefix is the Redis namespace. Short by design — there
// is one key per active user, so the keyspace cardinality is bounded
// by the number of authenticated callers per TTL window.
const userStateKeyPrefix = "user_state:"

// userStateCachedPayload is the JSON shape persisted in Redis. Kept
// intentionally separate from middleware.UserState so the wire format
// is decoupled from the in-memory type — adding fields to UserState
// later (e.g. EmailVerified) does not silently break backwards
// compatibility with cache entries written by older binaries during a
// rolling deploy.
type userStateCachedPayload struct {
	IsAdmin bool   `json:"is_admin"`
	Status  string `json:"status"`
}

// CachedUserStateChecker wraps an inner middleware.UserStateChecker
// (the postgres adapter, in production) with a Redis-backed hot
// cache. Implements middleware.UserStateChecker so the auth
// middleware never learns that caching exists.
//
// Cache semantics — match CachedSubscriptionReader so operators have
// one mental model for both:
//   - Hit: Redis value wins, no inner call.
//   - Miss: delegate, write the result back with TTL.
//   - Inner returns ErrUserNotFound: do NOT cache the negative
//     result — a user being recreated under the same id is not a
//     scenario we support, but caching "gone" would amplify any
//     transient lookup glitch into a 30s lockout.
//   - Redis read error: log + fall through to the inner reader so
//     the request keeps flowing.
//   - Redis write error: log, return the inner result — the caller
//     gets the authoritative answer even when the cache is broken.
type CachedUserStateChecker struct {
	client *goredis.Client
	inner  middleware.UserStateChecker
	ttl    time.Duration
}

// NewCachedUserStateChecker returns a Redis-fronted decorator over
// `inner`. Pass DefaultUserStateCacheTTL unless there is a strong
// reason to deviate (e.g. integration tests that need a shorter
// window).
func NewCachedUserStateChecker(
	client *goredis.Client,
	inner middleware.UserStateChecker,
	ttl time.Duration,
) *CachedUserStateChecker {
	if ttl <= 0 {
		ttl = DefaultUserStateCacheTTL
	}
	return &CachedUserStateChecker{
		client: client,
		inner:  inner,
		ttl:    ttl,
	}
}

// GetUserState satisfies middleware.UserStateChecker.
func (c *CachedUserStateChecker) GetUserState(
	ctx context.Context,
	userID uuid.UUID,
) (middleware.UserState, error) {
	key := userStateKeyPrefix + userID.String()

	// 1. Cache hit?
	val, err := c.client.Get(ctx, key).Result()
	if err == nil {
		var payload userStateCachedPayload
		jErr := json.Unmarshal([]byte(val), &payload)
		if jErr == nil {
			return middleware.UserState{
				IsAdmin: payload.IsAdmin,
				Status:  user.UserStatus(payload.Status),
			}, nil
		}
		// Corrupted entry — log and fall through, the next write will
		// fix it.
		slog.Warn("user state cache: malformed payload, refreshing",
			"user_id", userID, "error", jErr)
	} else if !errors.Is(err, goredis.Nil) {
		slog.Warn("user state cache: redis get failed, falling back to inner",
			"user_id", userID, "error", err)
	}

	// 2. Cache miss — consult the inner reader.
	state, ierr := c.inner.GetUserState(ctx, userID)
	if ierr != nil {
		// Do not cache errors. ErrUserNotFound included — see semantics
		// note in the type doc.
		return state, ierr
	}

	// 3. Write-through. Best-effort.
	payload, mErr := json.Marshal(userStateCachedPayload{
		IsAdmin: state.IsAdmin,
		Status:  string(state.Status),
	})
	if mErr != nil {
		// Marshalling our own struct should never fail, but if it does
		// we still return the authoritative answer to the caller.
		slog.Warn("user state cache: marshal failed",
			"user_id", userID, "error", mErr)
		return state, nil
	}
	if sErr := c.client.Set(ctx, key, payload, c.ttl).Err(); sErr != nil {
		slog.Warn("user state cache: redis set failed",
			"user_id", userID, "error", sErr)
	}
	return state, nil
}

// Invalidate evicts the cached entry for userID. Call this from any
// code path that mutates users.is_admin or users.status so the next
// authenticated request sees the new state immediately instead of
// waiting for the TTL. Missing key is NOT an error.
func (c *CachedUserStateChecker) Invalidate(
	ctx context.Context,
	userID uuid.UUID,
) error {
	key := userStateKeyPrefix + userID.String()
	_, err := c.client.Del(ctx, key).Result()
	return err
}
