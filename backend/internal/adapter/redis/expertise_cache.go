package redis

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"

	portservice "marketplace-backend/internal/port/service"
)

// DefaultExpertiseCacheTTL is 5 minutes. Expertise lists change very
// rarely (organizations set them once and edit infrequently), so we
// can afford a long TTL — the invalidation hook fired by the app
// service ensures stale data never lasts more than one round-trip
// after a write.
const DefaultExpertiseCacheTTL = 5 * time.Minute

// expertiseKeyPrefix namespaces every expertise entry. Short on
// purpose to keep Redis memory low — there can be one entry per
// organization and we expect O(active orgs) of them.
const expertiseKeyPrefix = "expertise:org:"

// CachedExpertiseReader wraps an inner ExpertiseReader (the app
// service) with a Redis-backed cache. Implements
// port/service.ExpertiseReader so the public profile and search
// decoration paths never learn that caching is in play.
//
// Cache semantics:
//   - Hit: the Redis value (JSON-encoded slice) wins, no DB read.
//   - Miss: delegate to inner, write the result back with TTL.
//   - Error reading from Redis: log + fall through to inner — a
//     degraded cache must never cause requests to fail.
//   - Error writing to Redis: log, return the inner result — the
//     caller must see the authoritative answer even if the cache
//     is broken.
//
// Stampede protection: a singleflight.Group coalesces concurrent
// misses so a single underlying call serves every concurrent
// caller. This is critical for very-hot paths (a public profile
// page that suddenly trends and triggers thousands of simultaneous
// reads in cache-cold state).
type CachedExpertiseReader struct {
	client *goredis.Client
	inner  portservice.ExpertiseReader
	ttl    time.Duration
	group  singleflight.Group
}

// NewCachedExpertiseReader returns a cache decorator over `inner`.
// Pass DefaultExpertiseCacheTTL unless you have a strong reason to
// deviate. A non-positive TTL is normalised to the default so
// integration tests that forget to set one still get a sane value.
func NewCachedExpertiseReader(client *goredis.Client, inner portservice.ExpertiseReader, ttl time.Duration) *CachedExpertiseReader {
	if ttl <= 0 {
		ttl = DefaultExpertiseCacheTTL
	}
	return &CachedExpertiseReader{
		client: client,
		inner:  inner,
		ttl:    ttl,
	}
}

// ListByOrganization satisfies port/service.ExpertiseReader. The
// Redis payload is the JSON encoding of the keys slice — a tiny
// marshal cost per miss but unambiguous reconstruction on hit
// (vs a comma-joined string that would mishandle commas in keys
// the day someone introduces them).
func (c *CachedExpertiseReader) ListByOrganization(ctx context.Context, orgID uuid.UUID) ([]string, error) {
	key := expertiseKeyPrefix + orgID.String()

	// 1. Cache hit?
	if hit, ok := c.tryGet(ctx, key); ok {
		return hit, nil
	}

	// 2. Cache miss — coalesce concurrent callers via singleflight
	//    so we hit the DB once even under a thundering herd.
	v, err, _ := c.group.Do(key, func() (any, error) {
		return c.fillFromInner(ctx, key, orgID)
	})
	if err != nil {
		return nil, err
	}
	keys, ok := v.([]string)
	if !ok || keys == nil {
		// Defensive: singleflight should have returned exactly what
		// fillFromInner emitted, but a future refactor could bypass
		// the assertion. Always hand back a non-nil slice.
		return []string{}, nil
	}
	return keys, nil
}

// tryGet returns (keys, true) on a clean hit, ([], false) on miss
// or transient cache error so the caller can fall back to inner.
func (c *CachedExpertiseReader) tryGet(ctx context.Context, key string) ([]string, bool) {
	raw, err := c.client.Get(ctx, key).Bytes()
	if err == nil {
		var keys []string
		jerr := json.Unmarshal(raw, &keys)
		if jerr == nil {
			if keys == nil {
				keys = []string{}
			}
			return keys, true
		}
		// Corrupt entry — log and treat as miss so we re-fetch and
		// overwrite with a fresh, valid encoding.
		slog.Warn("expertise cache: corrupt entry, treating as miss",
			"key", key, "error", jerr)
		return nil, false
	}
	if !errors.Is(err, goredis.Nil) {
		// Network / wire error — degrade gracefully.
		slog.Warn("expertise cache: redis get failed, falling back to inner",
			"key", key, "error", err)
	}
	return nil, false
}

// fillFromInner is the cache-miss path: consults the inner reader
// and writes the result back. Best-effort on the Set: a failed
// write never masks the authoritative answer.
func (c *CachedExpertiseReader) fillFromInner(ctx context.Context, key string, orgID uuid.UUID) ([]string, error) {
	keys, err := c.inner.ListByOrganization(ctx, orgID)
	if err != nil {
		return nil, err
	}
	if keys == nil {
		keys = []string{}
	}
	payload, jerr := json.Marshal(keys)
	if jerr != nil {
		// Should never happen for []string, but log so an unexpected
		// type substitution at the inner layer surfaces.
		slog.Warn("expertise cache: marshal failed, skipping cache write",
			"key", key, "error", jerr)
		return keys, nil
	}
	if sErr := c.client.Set(ctx, key, payload, c.ttl).Err(); sErr != nil {
		slog.Warn("expertise cache: redis set failed",
			"key", key, "error", sErr)
	}
	return keys, nil
}

// Invalidate removes the cached entry for orgID. Callers MUST
// invoke this after any successful SetExpertise so the next read
// reflects reality immediately instead of waiting for the TTL.
//
// Missing key is NOT an error — Del returns 0 and we treat it as
// a no-op, matching the CachedSubscriptionReader contract.
func (c *CachedExpertiseReader) Invalidate(ctx context.Context, orgID uuid.UUID) error {
	key := expertiseKeyPrefix + orgID.String()
	_, err := c.client.Del(ctx, key).Result()
	return err
}
