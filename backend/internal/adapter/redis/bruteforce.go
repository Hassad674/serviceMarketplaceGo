package redis

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// Brute-force protection constants. Centralised here so the policy is
// reviewable in a single place rather than hidden inside multiple
// inline literals.
const (
	// MaxLoginAttempts is the threshold at which a sliding 15-min
	// window of failed logins triggers the lockout flag.
	MaxLoginAttempts = 5

	// LoginAttemptWindow is the TTL of the per-email counter.
	LoginAttemptWindow = 15 * time.Minute

	// LoginLockoutDuration is how long the lockout flag persists once
	// MaxLoginAttempts has been hit.
	LoginLockoutDuration = 30 * time.Minute

	loginAttemptsKeyPrefix = "login_attempts:"
	loginLockedKeyPrefix   = "login_locked:"
)

// recordFailureScript atomically increments the attempts counter,
// initialises its TTL on the first failure, and — when the counter
// reaches the lockout threshold — sets the lockout flag with its own
// TTL. Doing all three in one round-trip removes the
// INCR-EXPIRE-SET-EXPIRE race that a naive Go-side implementation
// would have.
//
// KEYS[1]: attempts counter
// KEYS[2]: lockout flag
// ARGV[1]: attempts window seconds
// ARGV[2]: lockout duration seconds
// ARGV[3]: max attempts
//
// Return: 1 if the lockout flag was just set, 0 otherwise.
var recordFailureScript = goredis.NewScript(`
local count = redis.call('INCR', KEYS[1])
if count == 1 then
	redis.call('EXPIRE', KEYS[1], ARGV[1])
end
if count >= tonumber(ARGV[3]) then
	redis.call('SET', KEYS[2], '1', 'EX', ARGV[2])
	return 1
end
return 0
`)

// BruteForceService is the Redis-backed implementation of
// service.BruteForceService.
//
// Storage layout per email:
//   - login_attempts:<email>   counter, TTL 15min
//   - login_locked:<email>     "1", TTL 30min, set when counter >= 5
//
// Email is normalised at the call boundary (lowercased + trimmed) so
// a typed-with-uppercase email cannot be a separate counter from the
// same address typed in lowercase.
type BruteForceService struct {
	client            *goredis.Client
	maxAttempts       int
	attemptsWindow    time.Duration
	lockoutDuration   time.Duration
}

func NewBruteForceService(client *goredis.Client) *BruteForceService {
	return &BruteForceService{
		client:          client,
		maxAttempts:     MaxLoginAttempts,
		attemptsWindow:  LoginAttemptWindow,
		lockoutDuration: LoginLockoutDuration,
	}
}

// NewBruteForceServiceWithPolicy lets tests override the default
// policy without touching package-level constants. Production callers
// should use NewBruteForceService.
func NewBruteForceServiceWithPolicy(
	client *goredis.Client,
	maxAttempts int,
	attemptsWindow, lockoutDuration time.Duration,
) *BruteForceService {
	return &BruteForceService{
		client:          client,
		maxAttempts:     maxAttempts,
		attemptsWindow:  attemptsWindow,
		lockoutDuration: lockoutDuration,
	}
}

func normaliseEmail(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

func (s *BruteForceService) attemptsKey(email string) string {
	return loginAttemptsKeyPrefix + email
}

func (s *BruteForceService) lockedKey(email string) string {
	return loginLockedKeyPrefix + email
}

// IsLocked returns true when the lockout flag exists for the given
// email. A Redis failure returns (false, err) so the caller can choose
// to fail open (we do — see brute force on the auth handler).
func (s *BruteForceService) IsLocked(ctx context.Context, email string) (bool, error) {
	email = normaliseEmail(email)
	if email == "" {
		return false, nil
	}
	count, err := s.client.Exists(ctx, s.lockedKey(email)).Result()
	if err != nil {
		return false, fmt.Errorf("brute force is_locked: %w", err)
	}
	return count > 0, nil
}

// RecordFailure runs the atomic Lua script that bumps the counter and,
// on the threshold-hit step, sets the lockout flag. The boolean
// return value of the script (1 = lockout just triggered) is not
// surfaced — callers do not need it because IsLocked covers the same
// information on the next attempt.
func (s *BruteForceService) RecordFailure(ctx context.Context, email string) error {
	email = normaliseEmail(email)
	if email == "" {
		return nil
	}
	keys := []string{s.attemptsKey(email), s.lockedKey(email)}
	args := []interface{}{
		int(s.attemptsWindow.Seconds()),
		int(s.lockoutDuration.Seconds()),
		s.maxAttempts,
	}
	if _, err := recordFailureScript.Run(ctx, s.client, keys, args...).Int(); err != nil {
		return fmt.Errorf("brute force record_failure: %w", err)
	}
	return nil
}

// RecordSuccess deletes both keys atomically. Using a pipeline keeps
// the round-trip count down (1 instead of 2) without needing another
// Lua script — DEL is idempotent on missing keys so the pipeline
// cannot fail asymmetrically.
func (s *BruteForceService) RecordSuccess(ctx context.Context, email string) error {
	email = normaliseEmail(email)
	if email == "" {
		return nil
	}
	pipe := s.client.Pipeline()
	pipe.Del(ctx, s.attemptsKey(email))
	pipe.Del(ctx, s.lockedKey(email))
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("brute force record_success: %w", err)
	}
	return nil
}

// RetryAfter returns the TTL of the lockout flag, or zero when the
// email is not locked. Returns ErrNoLockout on a clean state so
// callers that only need a duration can return zero unchanged.
func (s *BruteForceService) RetryAfter(ctx context.Context, email string) (time.Duration, error) {
	email = normaliseEmail(email)
	if email == "" {
		return 0, nil
	}
	ttl, err := s.client.TTL(ctx, s.lockedKey(email)).Result()
	if err != nil {
		return 0, fmt.Errorf("brute force retry_after: %w", err)
	}
	// Redis returns -2 for "key does not exist" and -1 for "key has no
	// TTL" (which our setters never produce, but defensive code is
	// cheap). Treat both as unlocked.
	if ttl < 0 {
		return 0, nil
	}
	return ttl, nil
}

// ErrNoLockout is returned by RetryAfter when no lockout flag exists.
// Currently unused (we return zero instead) but exported so a future
// caller can branch on it explicitly without re-checking IsLocked.
var ErrNoLockout = errors.New("no lockout in effect")
