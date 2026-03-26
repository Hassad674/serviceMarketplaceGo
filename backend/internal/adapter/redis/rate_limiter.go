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

// rateLimitScript atomically increments the counter and sets the TTL on first increment.
// This eliminates the race condition between INCR and EXPIRE in the non-atomic version.
var rateLimitScript = goredis.NewScript(`
local c = redis.call('INCR', KEYS[1])
if c == 1 then
	redis.call('EXPIRE', KEYS[1], ARGV[1])
end
return c
`)

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
	windowSec := int(messagingRateWindow.Seconds())

	count, err := rateLimitScript.Run(ctx, rl.client, []string{key}, windowSec).Int64()
	if err != nil {
		return false, fmt.Errorf("rate limit check: %w", err)
	}

	return count <= messagingRateLimit, nil
}
