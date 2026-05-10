package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/twofactor"
)

// ErrTwoFactorChallengeNotFound is returned by FindLatestPendingForUser
// when no pending challenge row exists. Adapters surface this sentinel
// so the app layer can map it to twofactor.ErrChallengeNotFound without
// importing sql.ErrNoRows directly.
var ErrTwoFactorChallengeNotFound = errors.New("two_factor_challenge: not found")

// TwoFactorChallengeRepository persists and queries challenge rows.
// The interface stays narrow on purpose — only the four operations the
// app service needs (no admin sweep, no list, no purge) — so the mock
// in tests is trivial and a future Redis-backed adapter does not need
// to implement methods it would no-op anyway.
type TwoFactorChallengeRepository interface {
	// Create persists a new challenge. The row is expected to be unique
	// per (id) — there is no upsert semantic. Implementations return a
	// wrapped error including the operation context.
	Create(ctx context.Context, c *twofactor.Challenge) error

	// FindLatestPendingForUser returns the most recent challenge that is
	// neither used nor expired for the given user. Returns
	// ErrTwoFactorChallengeNotFound when no such row exists. The "latest"
	// ordering is by created_at DESC so a user who hammered "Resend
	// code" still verifies against the freshest issuance.
	FindLatestPendingForUser(ctx context.Context, userID uuid.UUID) (*twofactor.Challenge, error)

	// MarkUsed flips used_at to NOW() for the given challenge id. Idempotent
	// — calling on an already-used row is a no-op. Returns
	// ErrTwoFactorChallengeNotFound when the id does not match any row.
	MarkUsed(ctx context.Context, id uuid.UUID) error

	// DecrementAttempts subtracts one from attempts_left, floored at 0.
	// Used after a code mismatch so a brute-force attacker cannot keep
	// hammering the same challenge. Returns ErrTwoFactorChallengeNotFound
	// when the id does not match any row.
	DecrementAttempts(ctx context.Context, id uuid.UUID) error
}
