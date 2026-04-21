package redis

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"

	portservice "marketplace-backend/internal/port/service"
)

// DefaultSubscriptionCacheTTL is 60s — short enough that a state change
// (subscribe, cancel, past_due) surfaces quickly across the cluster, long
// enough that the cache absorbs the per-milestone-release hit path without
// touching Postgres on every single call.
const DefaultSubscriptionCacheTTL = 60 * time.Second

// subscriptionKeyPrefix is the Redis namespace. Kept short to minimise
// bytes-per-key since we expect O(active_users) of them.
const subscriptionKeyPrefix = "sub:active:"

// CachedSubscriptionReader wraps an inner SubscriptionReader (the app
// layer) with a Redis-backed hot cache. Implements
// port/service.SubscriptionReader so the payment service never learns
// that caching even exists.
//
// Cache semantics:
//   - Hit: the Redis value wins, no DB read.
//   - Miss: delegate to inner, write the result back with TTL.
//   - Error reading from Redis: log + fall through to inner — a degraded
//     cache must never cause requests to fail.
//   - Error writing to Redis: log, return the inner result — the caller
//     must see the authoritative answer even if the cache is broken.
type CachedSubscriptionReader struct {
	client *goredis.Client
	inner  portservice.SubscriptionReader
	ttl    time.Duration
}

// NewCachedSubscriptionReader returns a decorator over `inner` that
// caches per-user results in Redis for `ttl`. Pass DefaultSubscriptionCacheTTL
// unless you have a strong reason to deviate.
func NewCachedSubscriptionReader(client *goredis.Client, inner portservice.SubscriptionReader, ttl time.Duration) *CachedSubscriptionReader {
	if ttl <= 0 {
		ttl = DefaultSubscriptionCacheTTL
	}
	return &CachedSubscriptionReader{
		client: client,
		inner:  inner,
		ttl:    ttl,
	}
}

// IsActive satisfies port/service.SubscriptionReader.
func (c *CachedSubscriptionReader) IsActive(ctx context.Context, userID uuid.UUID) (bool, error) {
	key := subscriptionKeyPrefix + userID.String()

	// 1. Cache hit?
	val, err := c.client.Get(ctx, key).Result()
	if err == nil {
		return val == "1", nil
	}
	if !errors.Is(err, goredis.Nil) {
		// Network / wire error — log and fall through to inner so the
		// request path stays up.
		slog.Warn("subscription cache: redis get failed, falling back to inner",
			"user_id", userID, "error", err)
	}

	// 2. Cache miss — consult the inner reader.
	active, ierr := c.inner.IsActive(ctx, userID)
	if ierr != nil {
		return active, ierr
	}

	// 3. Write-through. Best-effort: a failed Set never masks the
	//    authoritative answer we already computed.
	payload := "0"
	if active {
		payload = "1"
	}
	if sErr := c.client.Set(ctx, key, payload, c.ttl).Err(); sErr != nil {
		slog.Warn("subscription cache: redis set failed",
			"user_id", userID, "error", sErr)
	}
	return active, nil
}

// Invalidate removes the cached entry for userID. Callers MUST invoke
// this after every state change that could flip IsActive (subscribe,
// cancel, past_due entry / exit, plan change) so the next read reflects
// reality immediately instead of waiting for the TTL.
//
// Missing key is NOT an error — Del returns 0 and we treat it as a no-op.
func (c *CachedSubscriptionReader) Invalidate(ctx context.Context, userID uuid.UUID) error {
	key := subscriptionKeyPrefix + userID.String()
	_, err := c.client.Del(ctx, key).Result()
	return err
}
