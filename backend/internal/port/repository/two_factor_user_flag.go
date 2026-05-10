package repository

import (
	"context"

	"github.com/google/uuid"
)

// TwoFactorUserFlagRepository is a focused, additive port that exposes
// the users.two_factor_email_enabled column without bloating the wide
// UserRepository contract.
//
// Why a separate port: the main UserRepository scans 25+ columns and
// adding two_factor_email_enabled to its column list would force every
// existing mock and test fixture to update. ISP says "narrow ports for
// narrow consumers" — the auth service only needs Read on the login
// path and Set on the enable/disable path, so a 2-method interface is
// the right grain.
//
// The postgres adapter satisfies BOTH UserRepository and this interface
// because it provides the union of their methods. main.go wires the
// same concrete repo to both ports.
type TwoFactorUserFlagRepository interface {
	// IsEmailTwoFactorEnabled returns the current value of
	// users.two_factor_email_enabled for the given user. Returns
	// (false, ErrUserNotFound) when the row is missing.
	IsEmailTwoFactorEnabled(ctx context.Context, userID uuid.UUID) (bool, error)

	// SetEmailTwoFactorEnabled flips the flag. Used by the
	// /me/two-factor/enable endpoint (after a confirmation challenge
	// is verified) and /me/two-factor/disable endpoint (after fresh
	// password re-auth).
	SetEmailTwoFactorEnabled(ctx context.Context, userID uuid.UUID, enabled bool) error
}
