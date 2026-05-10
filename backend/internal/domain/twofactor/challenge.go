// Package twofactor models the email-based two-factor authentication
// challenge that gates login when a user has opted in.
//
// The package is intentionally scoped to the challenge lifecycle —
// generation, validation, expiry, attempt accounting. It knows nothing
// about emails, HTTP, or persistence. Those concerns live in the
// adapter (postgres) and app (twofactor service) layers.
//
// Invariants enforced here:
//
//   - A challenge is created with a positive number of attempts and a
//     future expiry. Constructors reject zero-attempt or already-expired
//     inputs at build time so a buggy caller cannot persist a stillborn
//     row.
//   - A challenge is "pending" when it is neither used (UsedAt nil) nor
//     expired (ExpiresAt > now). The Pending() helper centralises the
//     check so handlers cannot drift apart.
//   - The 6-digit code is generated via crypto/rand in code.go and
//     never stored in plaintext — see service.RequestChallenge for the
//     bcrypt-hash-and-discard pattern.
package twofactor

import (
	"net"
	"time"

	"github.com/google/uuid"
)

// DefaultAttempts is the per-challenge attempt budget. Five is enough
// to absorb a typo or two and small enough to fall well under the
// brute-force horizon for a 6-digit code (1e6 / 5 = 200_000 challenges
// to exhaust the keyspace, which the rate limiter caps at < 1/min).
const DefaultAttempts = 5

// DefaultTTL is the wall-clock validity window of a challenge. Ten
// minutes covers a normal email-delivery latency tail (Resend p99
// roughly 30s, slow providers up to 2-3 min) plus user reaction time
// without leaving a stale row exploitable for hours.
const DefaultTTL = 10 * time.Minute

// Challenge is the persisted record of a single 2FA attempt sent to
// a user's email. It is ROW-IMMUTABLE except for AttemptsLeft (which
// only ever decreases) and UsedAt (set at most once when the matching
// code is verified). All construction goes through New() so the
// invariants below hold for every persisted row.
type Challenge struct {
	ID            uuid.UUID
	UserID        uuid.UUID
	CodeHash      string
	AttemptsLeft  int
	ExpiresAt     time.Time
	UsedAt        *time.Time
	CreatedAt     time.Time
	ClientIP      *net.IP
	UserAgentHash string
}

// NewChallengeInput groups the constructor arguments. Using a struct
// keeps the parameter list under the project's 4-arg ceiling and lets
// us extend the input shape (e.g. extra forensic fields) without a
// breaking change at every call site.
type NewChallengeInput struct {
	UserID        uuid.UUID
	CodeHash      string
	AttemptsLeft  int    // 0 falls back to DefaultAttempts
	TTL           time.Duration // 0 falls back to DefaultTTL
	ClientIP      string // raw IP, parsed and stored as net.IP; "" → nil
	UserAgentHash string // SHA-256 first 16 hex chars, opaque to this package
}

// New validates the input and returns a fresh Challenge ready to be
// persisted. The created_at and expires_at timestamps are computed
// from time.Now() so callers never need to thread a clock through the
// constructor — tests that need deterministic timestamps can stub
// time.Now via a global override (see clock.go).
func New(in NewChallengeInput) (*Challenge, error) {
	if in.UserID == uuid.Nil {
		return nil, ErrUserIDRequired
	}
	if in.CodeHash == "" {
		return nil, ErrCodeHashRequired
	}

	attempts := in.AttemptsLeft
	if attempts <= 0 {
		attempts = DefaultAttempts
	}
	ttl := in.TTL
	if ttl <= 0 {
		ttl = DefaultTTL
	}

	var ip *net.IP
	if in.ClientIP != "" {
		parsed := net.ParseIP(in.ClientIP)
		if parsed != nil {
			ip = &parsed
		}
	}

	now := nowFn()
	return &Challenge{
		ID:            uuid.New(),
		UserID:        in.UserID,
		CodeHash:      in.CodeHash,
		AttemptsLeft:  attempts,
		ExpiresAt:     now.Add(ttl),
		UsedAt:        nil,
		CreatedAt:     now,
		ClientIP:      ip,
		UserAgentHash: in.UserAgentHash,
	}, nil
}

// IsExpired reports whether the challenge has passed its expiry. The
// comparison is strict (>), so a challenge expiring exactly at "now"
// is still valid for the in-flight verification — easier to reason
// about than "now ≥ expiry" and aligns with the Postgres comparison
// the adapter would use.
func (c *Challenge) IsExpired() bool {
	return nowFn().After(c.ExpiresAt)
}

// IsUsed reports whether the challenge has already been redeemed.
// Once UsedAt is set, the row is effectively terminal — no further
// verification can succeed against it.
func (c *Challenge) IsUsed() bool {
	return c.UsedAt != nil
}

// IsPending returns true when the challenge is still verifiable —
// neither expired nor used and with at least one attempt left. This
// is the dominant predicate the verify path uses, so centralising it
// avoids subtle drift between handlers.
func (c *Challenge) IsPending() bool {
	return !c.IsUsed() && !c.IsExpired() && c.AttemptsLeft > 0
}

// MarkUsed stamps UsedAt with the current time. The caller is expected
// to persist the change via the repository; the domain object only
// transitions the in-memory state.
func (c *Challenge) MarkUsed() {
	now := nowFn()
	c.UsedAt = &now
}

// DecrementAttempts subtracts one from AttemptsLeft, floored at 0. The
// caller persists the new value; the domain only enforces the floor
// so a buggy adapter cannot drive the counter negative.
func (c *Challenge) DecrementAttempts() {
	if c.AttemptsLeft > 0 {
		c.AttemptsLeft--
	}
}
