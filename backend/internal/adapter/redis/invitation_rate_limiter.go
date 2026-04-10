package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

// Invitation rate limit: at most N invitations per org per window.
// The window is deliberately an hour so a busy Admin setting up a new
// team can invite ~10 people in a batch without hitting the cap, while
// still preventing automated abuse.
const (
	invitationRateLimit  = 10
	invitationRateWindow = 1 * time.Hour
)

// InvitationRateLimiter enforces a per-organization rate limit on the
// number of team invitations that can be sent in a rolling hourly window.
//
// Implementation mirrors MessagingRateLimiter: a single Redis INCR with
// a TTL on first increment. The atomic Lua script (rateLimitScript,
// defined in rate_limiter.go) removes the race between INCR and EXPIRE.
//
// The key is scoped to the organization, not the user, because the
// limit protects the org from bulk-send abuse regardless of which
// Owner/Admin actually sent the invitation.
type InvitationRateLimiter struct {
	client *goredis.Client
}

func NewInvitationRateLimiter(client *goredis.Client) *InvitationRateLimiter {
	return &InvitationRateLimiter{client: client}
}

func invitationRateLimitKey(orgID uuid.UUID) string {
	return "ratelimit:invitations:" + orgID.String()
}

// Allow increments the counter for the given org and returns whether
// the current request is within the window's cap. Returns
// (allowed bool, err error) — on Redis failure, err is non-nil and
// allowed is false (fail-closed, safer default).
func (rl *InvitationRateLimiter) Allow(ctx context.Context, orgID uuid.UUID) (bool, error) {
	key := invitationRateLimitKey(orgID)
	windowSec := int(invitationRateWindow.Seconds())

	count, err := rateLimitScript.Run(ctx, rl.client, []string{key}, windowSec).Int64()
	if err != nil {
		return false, fmt.Errorf("invitation rate limit check: %w", err)
	}
	return count <= invitationRateLimit, nil
}
