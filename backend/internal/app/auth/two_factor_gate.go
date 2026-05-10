package auth

import (
	"context"

	"github.com/google/uuid"
)

// TwoFactorGate is the narrow surface the auth service uses to gate
// the login flow on email 2FA. Defined here (not in port/) because it
// describes a same-layer collaboration between two app services
// (auth ↔ twofactor) — the twofactor service implements this
// indirectly through a thin adapter struct in cmd/api/main.go.
//
// A nil gate is allowed: the auth service treats it as "feature not
// wired" and skips the 2FA branch entirely. This keeps unit tests
// that don't exercise 2FA simple — they pass nil and login works
// exactly like it did before B.6.1.
type TwoFactorGate interface {
	// IsEnabledForUser returns true when the given user has opted into
	// email 2FA. Backed by the users.two_factor_email_enabled column.
	IsEnabledForUser(ctx context.Context, userID uuid.UUID) (bool, error)

	// RequestChallenge generates a fresh 6-digit code, persists the
	// bcrypt hash, and emails the plaintext to the user. Returns a
	// stable challenge id so the caller can echo it to the client for
	// logging / future "resend" flows.
	RequestChallenge(ctx context.Context, in TwoFactorChallengeRequest) (challengeID uuid.UUID, err error)

	// VerifyChallenge checks the latest pending challenge against the
	// supplied code. Returns nil on success, twofactor.ErrChallengeNotFound
	// /ErrChallengeExpired/ErrAttemptsExhausted/ErrCodeMismatch on the
	// failure modes the handler maps to user-facing errors.
	VerifyChallenge(ctx context.Context, userID uuid.UUID, code string) error
}

// TwoFactorChallengeRequest is the auth-side view of the challenge
// request. The fields mirror the twofactor service's input so the
// adapter is a 1:1 mapping with no business logic.
type TwoFactorChallengeRequest struct {
	UserID        uuid.UUID
	EmailTo       string
	ClientIP      string
	UserAgentHash string
}
