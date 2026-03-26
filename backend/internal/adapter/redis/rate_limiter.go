package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

const (
	messagingRateLimit  = 60
	messagingRateWindow = 60 * time.Second
)

type MessagingRateLimiter struct {
	client *goredis.Client
}

func NewMessagingRateLimiter(client *goredis.Client) *MessagingRateLimiter {
	return &MessagingRateLimiter{client: client}
}

func rateLimitKey(userID uuid.UUID) string {
	return "ratelimit:messaging:" + userID.String()
}

func (rl *MessagingRateLimiter) Allow(ctx context.Context, userID uuid.UUID) (bool, error) {
	key := rateLimitKey(userID)

	count, err := rl.client.Incr(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("increment rate limit: %w", err)
	}

	// Set expiry on first increment
	if count == 1 {
		rl.client.Expire(ctx, key, messagingRateWindow)
	}

	return count <= messagingRateLimit, nil
}
