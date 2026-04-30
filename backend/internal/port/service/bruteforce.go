package service

import (
	"context"
	"time"
)

// BruteForceService throttles authentication attempts per email address
// to prevent online password guessing. The contract is intentionally
// thin (4 methods) so handlers can wrap any login-like flow without
// pulling in a heavier rate-limiting framework:
//
//	if locked, _ := bf.IsLocked(ctx, email); locked {
//	    retry, _ := bf.RetryAfter(ctx, email)
//	    response.Error(w, 429, "too_many_attempts", retry)
//	    return
//	}
//	// validate credentials...
//	if invalid {
//	    bf.RecordFailure(ctx, email)
//	    return
//	}
//	bf.RecordSuccess(ctx, email)
//
// Implementations track two pieces of state per email:
//   - a 15-minute rolling counter ("login_attempts:<email>")
//   - a 30-minute lockout flag set once the counter crosses 5
//     ("login_locked:<email>")
//
// On RecordSuccess BOTH keys are deleted so a successful login wipes
// the slate. Concurrency: every method must be atomic against
// concurrent callers, both within and across processes — that is why
// the production adapter uses a Lua script for INCR + EXPIRE +
// optional SET and never touches the keys from Go.
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
}
