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

	"marketplace-backend/internal/domain/freelanceprofile"
	"marketplace-backend/internal/port/repository"
)

// PublicFreelanceProfileReader is the local interface the cache
// adapter expects from the underlying service. Defined here (not
// in port/service) because repository.FreelanceProfileView lives
// in port/repository which transitively imports domain/moderation
// which imports port/service — moving the interface to
// port/service would create an import cycle. Keeping the
// interface co-located with the cache is the cleanest fix and
// has the side benefit of bundling the contract with its only
// consumer.
//
// The concrete *freelanceprofile.Service satisfies this contract
// by definition. Tests inject lightweight stubs.
type PublicFreelanceProfileReader interface {
	GetPublicByOrgID(ctx context.Context, orgID uuid.UUID) (*repository.FreelanceProfileView, error)
}

// freelanceProfileKeyPrefix namespaces every public freelance
// profile entry. Short on purpose — there is one entry per
// (cached) provider_personal org so memory scales linearly with
// active personas.
const freelanceProfileKeyPrefix = "profile:freelance:"

// CachedPublicFreelanceProfileReader wraps an inner
// PublicFreelanceProfileReader (the freelanceprofile app service)
// with a Redis-backed cache. Mirrors the agency-side
// CachedPublicProfileReader contract:
//
//   - Hit (real view): JSON value wins, no DB read.
//   - Hit (negative marker): translate back to
//     freelanceprofile.ErrProfileNotFound.
//   - Miss: delegate to inner. On success → write JSON with TTL.
//     On freelanceprofile.ErrProfileNotFound → write the negative
//     marker with the shorter negativeTTL.
//   - Inner error other than ErrProfileNotFound → bubble up, do
//     NOT cache.
//   - Redis read/write errors are logged + degraded.
//
// Stampede protection via singleflight.Group coalesces concurrent
// misses. Critical for viral-profile scenarios.
type CachedPublicFreelanceProfileReader struct {
	client      *goredis.Client
	inner       PublicFreelanceProfileReader
	ttl         time.Duration
	negativeTTL time.Duration
	group       singleflight.Group
}

// NewCachedPublicFreelanceProfileReader wires the cache decorator.
// Pass DefaultPublicProfileCacheTTL / DefaultPublicProfileNegativeTTL
// (defined in profile_cache.go) unless an integration test needs
// shorter values. Non-positive TTLs fall back to the defaults so
// half-configured wiring still produces a working cache.
func NewCachedPublicFreelanceProfileReader(client *goredis.Client, inner PublicFreelanceProfileReader, ttl, negativeTTL time.Duration) *CachedPublicFreelanceProfileReader {
	if ttl <= 0 {
		ttl = DefaultPublicProfileCacheTTL
	}
	if negativeTTL <= 0 {
		negativeTTL = DefaultPublicProfileNegativeTTL
	}
	return &CachedPublicFreelanceProfileReader{
		client:      client,
		inner:       inner,
		ttl:         ttl,
		negativeTTL: negativeTTL,
	}
}

// GetPublicByOrgID satisfies port/service.PublicFreelanceProfileReader.
//
// Cache payload format mirrors the agency cache: '{' prefix means
// JSON-encoded view, '_' marker means negative.
func (c *CachedPublicFreelanceProfileReader) GetPublicByOrgID(ctx context.Context, orgID uuid.UUID) (*repository.FreelanceProfileView, error) {
	key := freelanceProfileKeyPrefix + orgID.String()

	if hit, found, isNotFound := c.tryGet(ctx, key); found {
		if isNotFound {
			return nil, freelanceprofile.ErrProfileNotFound
		}
		return hit, nil
	}

	v, err, _ := c.group.Do(key, func() (any, error) {
		return c.fillFromInner(ctx, key, orgID)
	})
	if err != nil {
		return nil, err
	}
	view, _ := v.(*repository.FreelanceProfileView)
	return view, nil
}

// tryGet returns (view, found, isNotFound). See profile_cache.go
// for the discriminator-byte rationale.
func (c *CachedPublicFreelanceProfileReader) tryGet(ctx context.Context, key string) (*repository.FreelanceProfileView, bool, bool) {
	raw, err := c.client.Get(ctx, key).Bytes()
	if err == nil {
		if len(raw) == 1 && raw[0] == negativeMarker[0] {
			return nil, true, true
		}
		var view repository.FreelanceProfileView
		if jerr := json.Unmarshal(raw, &view); jerr == nil {
			return &view, true, false
		} else {
			slog.Warn("freelance profile cache: corrupt entry, treating as miss",
				"key", key, "error", jerr)
			return nil, false, false
		}
	}
	if !errors.Is(err, goredis.Nil) {
		slog.Warn("freelance profile cache: redis get failed, falling back to inner",
			"key", key, "error", err)
	}
	return nil, false, false
}

func (c *CachedPublicFreelanceProfileReader) fillFromInner(ctx context.Context, key string, orgID uuid.UUID) (*repository.FreelanceProfileView, error) {
	view, err := c.inner.GetPublicByOrgID(ctx, orgID)
	if err != nil {
		if errors.Is(err, freelanceprofile.ErrProfileNotFound) {
			c.writeNegative(ctx, key)
		}
		return nil, err
	}
	c.writePositive(ctx, key, view)
	return view, nil
}

func (c *CachedPublicFreelanceProfileReader) writePositive(ctx context.Context, key string, view *repository.FreelanceProfileView) {
	payload, jerr := json.Marshal(view)
	if jerr != nil {
		slog.Warn("freelance profile cache: marshal failed, skipping cache write",
			"key", key, "error", jerr)
		return
	}
	if sErr := c.client.Set(ctx, key, payload, c.ttl).Err(); sErr != nil {
		slog.Warn("freelance profile cache: redis set failed",
			"key", key, "error", sErr)
	}
}

func (c *CachedPublicFreelanceProfileReader) writeNegative(ctx context.Context, key string) {
	if sErr := c.client.Set(ctx, key, negativeMarker, c.negativeTTL).Err(); sErr != nil {
		slog.Warn("freelance profile cache: redis set negative failed",
			"key", key, "error", sErr)
	}
}

// Invalidate removes the cached entry for orgID. Callers MUST
// invoke this after every UpdateCore / UpdateAvailability /
// UpdateExpertise / UpdateVideo on a freelance profile so the
// next public read reflects reality immediately.
func (c *CachedPublicFreelanceProfileReader) Invalidate(ctx context.Context, orgID uuid.UUID) error {
	key := freelanceProfileKeyPrefix + orgID.String()
	_, err := c.client.Del(ctx, key).Result()
	return err
}
