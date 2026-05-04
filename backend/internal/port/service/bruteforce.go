package service

import (
	"context"
	"time"
)

// BruteForceService throttles authentication attempts per email address
// AND per source IP to prevent online password guessing. The contract
// has two parallel surfaces:
//
//   - per-EMAIL (Is/Record/Retry): credential-stuffing protection.
//     Tighter threshold (5/15min → 30min lock). Stops attackers with
//     a list of leaked emails from grinding each one.
//   - per-IP (IsIPLocked / RecordIPFailure / RetryAfterIP): DoS
//     protection. Looser threshold (20/15min → 60min lock). Higher
//     than per-email because shared NATs / corporate networks
//     legitimately produce more login traffic from one IP. Stops a
//     single attacker from locking N victim emails out by exhausting
//     the per-email counter on each.
//
// Both gates are checked at the start of Login; recording a failure
// hits both counters in lockstep. The two surfaces are deliberately
// independent — closing the email gate must not affect IP state and
// vice versa — so a single attacker can DoS at most their own IP.
//
// Implementations track:
//   - a 15-minute rolling counter ("login_attempts:<email>")
//   - a 30-minute lockout flag set once the counter crosses 5
//     ("login_locked:<email>")
//   - a 15-minute rolling IP counter ("login_attempts_ip:<ip>")
//   - a 60-minute IP lockout flag set once the counter crosses 20
//     ("login_locked_ip:<ip>")
//
// On RecordSuccess BOTH email keys are deleted so a successful login
// wipes the slate. The IP counters are NOT cleared on success — a
// shared-NAT user who guesses their password on attempt #18 should
// not unlock the gate for a co-located attacker. Concurrency: every
// method must be atomic against concurrent callers, both within and
// across processes — that is why the production adapter uses a Lua
// script for INCR + EXPIRE + optional SET and never touches the keys
// from Go.
type BruteForceService interface {
	// IsLocked reports whether the supplied email is currently locked
	// out. An empty email is treated as not-locked. A storage failure
	// returns (false, err) — fail open is safer than fail closed for
	// this read because we do not want a Redis blip to block every
	// login on the platform.
	IsLocked(ctx context.Context, email string) (bool, error)

	// RecordFailure increments the per-email attempt counter. When the
	// counter reaches the lockout threshold the implementation MUST
	// also set the lockout flag in the same atomic operation. The
	// caller never needs to react to RecordFailure errors — they are
	// logged at WARN by the implementation.
	RecordFailure(ctx context.Context, email string) error

	// RecordSuccess clears both the attempt counter and the lockout
	// flag for the given email. Always called on a successful login
	// so a user who finally typed the right password gets a clean
	// slate, even if they had 4 failed attempts in the same window.
	RecordSuccess(ctx context.Context, email string) error

	// RetryAfter returns the time-to-wait before the email is unlocked.
	// Returns 0 when the email is not locked. Used to populate the
	// Retry-After header on 429 responses.
	RetryAfter(ctx context.Context, email string) (time.Duration, error)

	// IsIPLocked reports whether the source IP has hit the per-IP
	// brute-force lockout. Empty IP is treated as not-locked. A
	// storage failure returns (false, err) so the caller can decide
	// fail-open vs fail-closed (matches IsLocked semantics).
	//
	// N4: this gate is the DoS counter-measure layered on top of the
	// per-email gate. The threshold is intentionally higher than per
	// email (20 vs 5) so shared NATs / corporate networks are not
	// over-throttled.
	IsIPLocked(ctx context.Context, ip string) (bool, error)

	// RecordIPFailure increments the per-IP failure counter and, on
	// reaching the lockout threshold, sets the IP lockout flag. Called
	// alongside RecordFailure on every failed login. Empty IP is a
	// no-op so callers do not need to check for unparseable
	// RemoteAddr at the call site.
	RecordIPFailure(ctx context.Context, ip string) error

	// RetryAfterIP returns the IP lockout TTL, or zero when the IP is
	// not locked. Used to populate the Retry-After header on 429
	// responses originating from the IP gate (typically larger than
	// the per-email TTL — 60 vs 30 minutes by default).
	RetryAfterIP(ctx context.Context, ip string) (time.Duration, error)
}
