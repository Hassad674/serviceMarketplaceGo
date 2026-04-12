package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

// Role permissions change rate limit: at most N changes per org per
// 24-hour window. The cap is deliberately lenient (a full setup phase
// can require many tweaks) while still stopping an attacker who has
// compromised an Owner session from flipping permissions at machine
// speed.
const (
	rolePermChangesLimit  = 20
	rolePermChangesWindow = 24 * time.Hour
)

// RolePermissionsRateLimiter enforces a per-organization rate limit
// on the number of role permission edits that can be saved in a
// 24-hour rolling window.
//
// Implementation mirrors InvitationRateLimiter: the atomic Lua script
// in rate_limiter.go does an INCR + EXPIRE so the TTL is set exactly
// once (on the first change) without race conditions between INCR
// and EXPIRE.
//
// The key is scoped to the organization, not the user — the cap
// protects the org's permission matrix regardless of which Owner
// session triggered the changes.
type RolePermissionsRateLimiter struct {
	client *goredis.Client
}

func NewRolePermissionsRateLimiter(client *goredis.Client) *RolePermissionsRateLimiter {
	return &RolePermissionsRateLimiter{client: client}
}

func rolePermChangesKey(orgID uuid.UUID) string {
	return "ratelimit:role_perm_changes:" + orgID.String()
}

// Allow increments the counter for the given org and returns whether
// the current save is within the daily cap. Fail-closed on Redis
// errors: a broken Redis must not let an attacker bypass the limit.
func (rl *RolePermissionsRateLimiter) Allow(ctx context.Context, orgID uuid.UUID) (bool, error) {
	key := rolePermChangesKey(orgID)
	windowSec := int(rolePermChangesWindow.Seconds())

	count, err := rateLimitScript.Run(ctx, rl.client, []string{key}, windowSec).Int64()
	if err != nil {
		return false, fmt.Errorf("role permissions rate limit check: %w", err)
	}
	return count <= rolePermChangesLimit, nil
}
