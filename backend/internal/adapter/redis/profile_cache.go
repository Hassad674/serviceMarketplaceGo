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

	"marketplace-backend/internal/domain/profile"
	portservice "marketplace-backend/internal/port/service"
)

// Cache TTLs are tuned per signal volatility:
//   - 60s for the hit / present case: short enough that operator-
//     visible edits surface within a minute even if the explicit
//     invalidate fails, long enough to absorb the read burst that
//     follows a profile being shared on social.
//   - 30s for the negative (not-found) case: 404 spam from broken
//     deep-links is the typical pattern; we want the second hit
//     to come from Redis but fast-forward back to the DB if the
//     org gets created in the meantime.
const (
	DefaultPublicProfileCacheTTL    = 60 * time.Second
	DefaultPublicProfileNegativeTTL = 30 * time.Second
)

// agencyProfileKeyPrefix is short on purpose — there is one entry
// per (cached) agency org, and we expect O(active orgs) of them.
const agencyProfileKeyPrefix = "profile:agency:"

// negativeMarker is the literal payload we write for a not-found
// entry. JSON encoding for a real profile starts with '{', so a
// single byte sentinel cannot collide with a valid hit payload.
// Keeping the marker as raw bytes (not a JSON null) avoids paying
// json.Unmarshal on the negative-cache hot path — this is the
// 404-flood scenario, where the whole point is to be cheap.
const negativeMarker = "_"

// CachedPublicProfileReader wraps an inner PublicProfileReader (the
// app service) with a Redis-backed cache. Implements
// port/service.PublicProfileReader so the public agency profile
// endpoint never learns that caching is in play.
//
// Cache semantics:
//   - Hit (real profile): Redis JSON value wins, no DB read.
//   - Hit (negative marker): translate back to
//     profile.ErrProfileNotFound so handler 404 logic still fires.
//   - Miss: delegate to inner. On success → write JSON with TTL.
//     On profile.ErrProfileNotFound → write the negative marker
//     with the shorter negativeTTL.
//   - Inner error other than ErrProfileNotFound → bubble up, do NOT
//     cache (a transient blip must not pin a 5xx for the full TTL).
//   - Redis read/write errors are logged + degraded — a degraded
//     cache must never cause requests to fail.
//
// Stampede protection: a singleflight.Group coalesces concurrent
// misses into a single underlying call so a viral profile that
// suddenly receives thousands of concurrent reads still hits the
// DB exactly once.
type CachedPublicProfileReader struct {
	client      *goredis.Client
	inner       portservice.PublicProfileReader
	keyPrefix   string
	ttl         time.Duration
	negativeTTL time.Duration
	group       singleflight.Group
}

// NewCachedPublicProfileReader returns the agency-scoped public
// profile cache. Pass DefaultPublicProfileCacheTTL /
// DefaultPublicProfileNegativeTTL unless an integration test
// requires shorter values. Non-positive TTLs are normalised to
// the defaults so test setup that forgets to set them still
// receives a sane value.
func NewCachedPublicProfileReader(client *goredis.Client, inner portservice.PublicProfileReader, ttl, negativeTTL time.Duration) *CachedPublicProfileReader {
	if ttl <= 0 {
		ttl = DefaultPublicProfileCacheTTL
	}
	if negativeTTL <= 0 {
		negativeTTL = DefaultPublicProfileNegativeTTL
	}
	return &CachedPublicProfileReader{
		client:      client,
		inner:       inner,
		keyPrefix:   agencyProfileKeyPrefix,
		ttl:         ttl,
		negativeTTL: negativeTTL,
	}
}

// GetProfile satisfies port/service.PublicProfileReader.
//
// The cache stores either a JSON-encoded *profile.Profile (the
// happy path) or the literal negativeMarker byte (the not-found
// path). The discriminator is the first byte: '{' for JSON, '_'
// for the marker — see the constant comment for why.
func (c *CachedPublicProfileReader) GetProfile(ctx context.Context, orgID uuid.UUID) (*profile.Profile, error) {
	key := c.keyPrefix + orgID.String()

	if hit, found, isNotFound := c.tryGet(ctx, key); found {
		if isNotFound {
			return nil, profile.ErrProfileNotFound
		}
		return hit, nil
	}

	v, err, _ := c.group.Do(key, func() (any, error) {
		return c.fillFromInner(ctx, key, orgID)
	})
	if err != nil {
		return nil, err
	}
	p, _ := v.(*profile.Profile)
	return p, nil
}

// tryGet returns the parsed cache value plus two flags:
//   - found:      true when the key exists in Redis (hit or negative)
//   - isNotFound: true when the key holds the negative marker
func (c *CachedPublicProfileReader) tryGet(ctx context.Context, key string) (*profile.Profile, bool, bool) {
	raw, err := c.client.Get(ctx, key).Bytes()
	if err == nil {
		if len(raw) == 1 && raw[0] == negativeMarker[0] {
			return nil, true, true
		}
		var p profile.Profile
		if jerr := json.Unmarshal(raw, &p); jerr == nil {
			return &p, true, false
		} else {
			// Corrupt entry — log and treat as miss so the next call
			// re-fetches and overwrites with a fresh, valid encoding.
			slog.Warn("profile cache: corrupt entry, treating as miss",
				"key", key, "error", jerr)
			return nil, false, false
		}
	}
	if !errors.Is(err, goredis.Nil) {
		slog.Warn("profile cache: redis get failed, falling back to inner",
			"key", key, "error", err)
	}
	return nil, false, false
}

// fillFromInner is the cache-miss path: consults the inner reader,
// writes the result back. Best-effort on the Set: a failed write
// never masks the authoritative answer the caller is waiting for.
func (c *CachedPublicProfileReader) fillFromInner(ctx context.Context, key string, orgID uuid.UUID) (*profile.Profile, error) {
	p, err := c.inner.GetProfile(ctx, orgID)
	if err != nil {
		// Negative-cache the not-found case so 404 spam is absorbed
		// at the cache layer. Other errors (DB blip, context cancel)
		// must NOT be cached: they are transient and we want the next
		// request to retry against the inner reader.
		if errors.Is(err, profile.ErrProfileNotFound) {
			c.writeNegative(ctx, key)
		}
		return nil, err
	}
	c.writePositive(ctx, key, p)
	return p, nil
}

func (c *CachedPublicProfileReader) writePositive(ctx context.Context, key string, p *profile.Profile) {
	payload, jerr := json.Marshal(p)
	if jerr != nil {
		slog.Warn("profile cache: marshal failed, skipping cache write",
			"key", key, "error", jerr)
		return
	}
	if sErr := c.client.Set(ctx, key, payload, c.ttl).Err(); sErr != nil {
		slog.Warn("profile cache: redis set failed",
			"key", key, "error", sErr)
	}
}

func (c *CachedPublicProfileReader) writeNegative(ctx context.Context, key string) {
	if sErr := c.client.Set(ctx, key, negativeMarker, c.negativeTTL).Err(); sErr != nil {
		slog.Warn("profile cache: redis set negative failed",
			"key", key, "error", sErr)
	}
}

// Invalidate removes the cached entry for orgID. Callers MUST
// invoke this after every state change that could flip the read
// answer (UpdateProfile / UpdateLocation / UpdateLanguages /
// UpdateAvailability) so the next read reflects reality
// immediately. Missing key is NOT an error — Del returns 0 and
// we treat it as a no-op.
func (c *CachedPublicProfileReader) Invalidate(ctx context.Context, orgID uuid.UUID) error {
	key := c.keyPrefix + orgID.String()
	_, err := c.client.Del(ctx, key).Result()
	return err
}
