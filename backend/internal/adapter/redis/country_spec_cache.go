package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"marketplace-backend/internal/domain/payment"
	"marketplace-backend/internal/port/service"
)

const (
	countrySpecKeyPrefix = "country_spec:"
	countrySpecTTL       = 24 * time.Hour
)

// CountrySpecCache implements service.CountrySpecService with Redis caching.
type CountrySpecCache struct {
	client *goredis.Client
	stripe service.StripeService
}

// NewCountrySpecCache creates a new CountrySpecCache.
func NewCountrySpecCache(client *goredis.Client, stripe service.StripeService) *CountrySpecCache {
	return &CountrySpecCache{client: client, stripe: stripe}
}

// GetFieldsForCountry returns specs from cache, falling back to Stripe on miss.
func (c *CountrySpecCache) GetFieldsForCountry(ctx context.Context, country string) (*payment.CountryFieldSpec, error) {
	spec, err := c.getFromCache(ctx, country)
	if err == nil && spec != nil {
		return spec, nil
	}

	// Cache miss: fetch from Stripe
	spec, err = c.stripe.GetCountrySpec(ctx, country)
	if err != nil {
		return nil, fmt.Errorf("fetch country spec: %w", err)
	}

	c.setCache(ctx, country, spec)
	return spec, nil
}

// WarmCache pre-loads all country specs from Stripe into Redis.
func (c *CountrySpecCache) WarmCache(ctx context.Context) error {
	specs, err := c.stripe.ListAllCountrySpecs(ctx)
	if err != nil {
		slog.Warn("failed to warm country spec cache", "error", err)
		return nil // graceful degradation
	}

	for _, spec := range specs {
		c.setCache(ctx, spec.Country, spec)
	}
	slog.Info("country spec cache warmed", "count", len(specs))
	return nil
}

func (c *CountrySpecCache) getFromCache(ctx context.Context, country string) (*payment.CountryFieldSpec, error) {
	key := countrySpecKeyPrefix + country
	val, err := c.client.Get(ctx, key).Result()
	if err == goredis.Nil {
		return nil, nil
	}
	if err != nil {
		slog.Warn("redis get country spec failed", "key", key, "error", err)
		return nil, nil // graceful degradation
	}

	var spec payment.CountryFieldSpec
	if err := json.Unmarshal([]byte(val), &spec); err != nil {
		return nil, fmt.Errorf("unmarshal cached spec: %w", err)
	}
	return &spec, nil
}

func (c *CountrySpecCache) setCache(ctx context.Context, country string, spec *payment.CountryFieldSpec) {
	key := countrySpecKeyPrefix + country
	data, err := json.Marshal(spec)
	if err != nil {
		slog.Warn("marshal country spec for cache", "error", err)
		return
	}
	if err := c.client.Set(ctx, key, data, countrySpecTTL).Err(); err != nil {
		slog.Warn("redis set country spec failed", "key", key, "error", err)
	}
}
