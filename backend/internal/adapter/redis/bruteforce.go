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
	// window of failed logins triggers the per-email lockout flag.
	MaxLoginAttempts = 5

	// LoginAttemptWindow is the TTL of the per-email counter.
	LoginAttemptWindow = 15 * time.Minute

	// LoginLockoutDuration is how long the per-email lockout flag
	// persists once MaxLoginAttempts has been hit.
	LoginLockoutDuration = 30 * time.Minute

	// MaxIPLoginAttempts is the threshold at which a sliding 15-min
	// window of failed logins from one source IP triggers the per-IP
	// lockout flag (N4). Set higher than per-email (20 vs 5) because
	// shared NATs / corporate networks legitimately produce more
	// login traffic from one IP than one user does. Set low enough
	// that an attacker cannot lock 100 victim emails out from a
	// single IP by spending only 5 wrong-password attempts each.
	MaxIPLoginAttempts = 20

	// IPLoginAttemptWindow is the TTL of the per-IP counter.
	IPLoginAttemptWindow = 15 * time.Minute

	// IPLoginLockoutDuration is how long the per-IP lockout flag
	// persists once MaxIPLoginAttempts has been hit. Longer than
	// per-email (60 vs 30 min) — a hostile IP that hits 20 wrong
	// passwords inside 15 minutes is almost certainly automated, and
	// the longer cool-down compounds the cost of running a botnet.
	IPLoginLockoutDuration = 60 * time.Minute

	loginAttemptsKeyPrefix   = "login_attempts:"
	loginLockedKeyPrefix     = "login_locked:"
	loginAttemptsIPKeyPrefix = "login_attempts_ip:"
	loginLockedIPKeyPrefix   = "login_locked_ip:"
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

	// IP-side policy (N4). Defaults to the package constants; tests
	// can override via NewBruteForceServiceWithIPPolicy.
	maxIPAttempts     int
	ipAttemptsWindow  time.Duration
	ipLockoutDuration time.Duration
}

func NewBruteForceService(client *goredis.Client) *BruteForceService {
	return &BruteForceService{
		client:            client,
		maxAttempts:       MaxLoginAttempts,
		attemptsWindow:    LoginAttemptWindow,
		lockoutDuration:   LoginLockoutDuration,
		maxIPAttempts:     MaxIPLoginAttempts,
		ipAttemptsWindow:  IPLoginAttemptWindow,
		ipLockoutDuration: IPLoginLockoutDuration,
	}
}

// NewBruteForceServiceWithPolicy lets tests override the default
// per-EMAIL policy without touching package-level constants. The IP
// policy stays at the package defaults — use
// NewBruteForceServiceWithIPPolicy when both knobs need tuning.
// Production callers should use NewBruteForceService.
func NewBruteForceServiceWithPolicy(
	client *goredis.Client,
	maxAttempts int,
	attemptsWindow, lockoutDuration time.Duration,
) *BruteForceService {
	return &BruteForceService{
		client:            client,
		maxAttempts:       maxAttempts,
		attemptsWindow:    attemptsWindow,
		lockoutDuration:   lockoutDuration,
		maxIPAttempts:     MaxIPLoginAttempts,
		ipAttemptsWindow:  IPLoginAttemptWindow,
		ipLockoutDuration: IPLoginLockoutDuration,
	}
}

// NewBruteForceServiceWithIPPolicy lets tests override the per-IP
// policy (N4) too. Used by the IP-gate tests so a 1-attempt threshold
// can exercise the lockout in a single line.
func NewBruteForceServiceWithIPPolicy(
	client *goredis.Client,
	maxAttempts int,
	attemptsWindow, lockoutDuration time.Duration,
	maxIPAttempts int,
	ipAttemptsWindow, ipLockoutDuration time.Duration,
) *BruteForceService {
	return &BruteForceService{
		client:            client,
		maxAttempts:       maxAttempts,
		attemptsWindow:    attemptsWindow,
		lockoutDuration:   lockoutDuration,
		maxIPAttempts:     maxIPAttempts,
		ipAttemptsWindow:  ipAttemptsWindow,
		ipLockoutDuration: ipLockoutDuration,
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

func (s *BruteForceService) attemptsIPKey(ip string) string {
	return loginAttemptsIPKeyPrefix + ip
}

func (s *BruteForceService) lockedIPKey(ip string) string {
	return loginLockedIPKeyPrefix + ip
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

// IsIPLocked returns true when the IP lockout flag exists for the
// given IP. A Redis failure returns (false, err) — same fail-open
// semantics as IsLocked. The string is the limiter-normalised key
// (rl.ClientIP) so an IPv6 attacker hopping inside a /64 hits the
// same bucket.
func (s *BruteForceService) IsIPLocked(ctx context.Context, ip string) (bool, error) {
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return false, nil
	}
	count, err := s.client.Exists(ctx, s.lockedIPKey(ip)).Result()
	if err != nil {
		return false, fmt.Errorf("brute force is_ip_locked: %w", err)
	}
	return count > 0, nil
}

// RecordIPFailure runs the same atomic Lua script that bumps the
// counter and sets the lockout flag on threshold-hit, but keyed on the
// IP namespace and parameterised with the IP-side policy. Empty IP is
// a no-op so callers do not have to check r.RemoteAddr at the call
// site.
func (s *BruteForceService) RecordIPFailure(ctx context.Context, ip string) error {
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return nil
	}
	keys := []string{s.attemptsIPKey(ip), s.lockedIPKey(ip)}
	args := []interface{}{
		int(s.ipAttemptsWindow.Seconds()),
		int(s.ipLockoutDuration.Seconds()),
		s.maxIPAttempts,
	}
	if _, err := recordFailureScript.Run(ctx, s.client, keys, args...).Int(); err != nil {
		return fmt.Errorf("brute force record_ip_failure: %w", err)
	}
	return nil
}

// RetryAfterIP returns the TTL of the IP lockout flag, or zero when
// the IP is not locked. Mirrors RetryAfter for the email side.
func (s *BruteForceService) RetryAfterIP(ctx context.Context, ip string) (time.Duration, error) {
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return 0, nil
	}
	ttl, err := s.client.TTL(ctx, s.lockedIPKey(ip)).Result()
	if err != nil {
		return 0, fmt.Errorf("brute force retry_after_ip: %w", err)
	}
	if ttl < 0 {
		return 0, nil
	}
	return ttl, nil
}

// ErrNoLockout is returned by RetryAfter when no lockout flag exists.
// Currently unused (we return zero instead) but exported so a future
// caller can branch on it explicitly without re-checking IsLocked.
var ErrNoLockout = errors.New("no lockout in effect")
