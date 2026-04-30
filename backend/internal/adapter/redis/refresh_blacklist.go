package redis

import (
	"context"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// refreshBlacklistKey is the Redis key prefix for blacklisted refresh
// token JTIs. Keep it narrow ("token_blacklist:") so a future
// access-token blacklist can use a different namespace without colliding.
const refreshBlacklistKey = "token_blacklist:"

// RefreshBlacklistService implements service.RefreshBlacklistService
// against a Redis 7 backend.
//
// Storage shape: SETEX token_blacklist:{jti} 1 <ttl_seconds>. The value
// "1" is a placeholder — only the existence of the key matters. We rely
// on Redis's per-key TTL to garbage-collect expired entries automatically
// so the blacklist can never grow unbounded across many rotations.
//
// Concurrency: SET with EX is atomic in Redis, so concurrent Add calls
// for the same jti are safe (the second call overwrites the first with
// the same value + a fresh TTL, which is harmless).
type RefreshBlacklistService struct {
	client *goredis.Client
}

func NewRefreshBlacklistService(client *goredis.Client) *RefreshBlacklistService {
	return &RefreshBlacklistService{client: client}
}

// Add stores the jti with the given ttl. Empty jti and non-positive ttl
// are no-ops to keep call sites free of defensive guards.
func (s *RefreshBlacklistService) Add(ctx context.Context, jti string, ttl time.Duration) error {
	if jti == "" || ttl <= 0 {
		return nil
	}
	if err := s.client.Set(ctx, refreshBlacklistKey+jti, "1", ttl).Err(); err != nil {
		return fmt.Errorf("refresh blacklist add: %w", err)
	}
	return nil
}

// Has reports whether the jti is currently blacklisted. Empty jti
// short-circuits to (false, nil) so callers can safely check tokens
// without a JTI claim. EXISTS is the cheapest probe and Redis returns
// 0 for both "key never set" and "key already expired" — both of which
// the caller treats identically.
func (s *RefreshBlacklistService) Has(ctx context.Context, jti string) (bool, error) {
	if jti == "" {
		return false, nil
	}
	count, err := s.client.Exists(ctx, refreshBlacklistKey+jti).Result()
	if err != nil {
		return false, fmt.Errorf("refresh blacklist has: %w", err)
	}
	return count > 0, nil
}
