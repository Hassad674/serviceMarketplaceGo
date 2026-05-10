package twofactor

import "errors"

// Sentinel errors emitted by the domain and bubbled up through the
// app and handler layers. Every public name carries an `Err` prefix
// per the project convention (mirrors domain/user/errors.go).
var (
	// ErrUserIDRequired is returned by New when called with a zero UUID.
	ErrUserIDRequired = errors.New("twofactor: user id is required")

	// ErrCodeHashRequired is returned by New when called with an empty
	// code hash. The caller is expected to bcrypt-hash the plaintext
	// code before calling New — never the reverse.
	ErrCodeHashRequired = errors.New("twofactor: code hash is required")

	// ErrChallengeNotFound is returned by the verify path when no
	// pending challenge exists for the user. The handler maps it to
	// 400 invalid_request because the client is supposed to call
	// /verify only after Login has returned requires_2fa=true.
	ErrChallengeNotFound = errors.New("twofactor: no pending challenge")

	// ErrChallengeExpired is returned when a pending challenge has
	// passed its expires_at. The handler maps it to 400 challenge_expired
	// so the client can prompt the user to request a fresh code.
	ErrChallengeExpired = errors.New("twofactor: challenge expired")

	// ErrAttemptsExhausted is returned when the attempts_left counter
	// has reached zero. The handler maps it to 429 too_many_attempts
	// so the client cannot keep brute-forcing the same row.
	ErrAttemptsExhausted = errors.New("twofactor: attempts exhausted")

	// ErrCodeMismatch is returned when bcrypt.Compare fails on a
	// pending challenge. The handler maps it to 400 invalid_code so
	// the user sees a clear "wrong code" message — the surrounding
	// rate limiter already prevents abuse.
	ErrCodeMismatch = errors.New("twofactor: code mismatch")
)
