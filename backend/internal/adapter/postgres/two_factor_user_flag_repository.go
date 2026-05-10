package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/user"
)

// ChallengeDB is the minimal *sql.DB surface the 2FA challenge
// adapter needs. Re-exported here so the cmd/api wiring layer can
// pass infra.DB without importing database/sql directly.
type ChallengeDB = *sql.DB

// TwoFactorFlagSetter is satisfied by *UserRepository — it pairs the
// IsEmail/SetEmail TwoFactorEnabled methods so the wiring layer can
// hold a narrow port reference instead of the wide UserRepository.
type TwoFactorFlagSetter interface {
	IsEmailTwoFactorEnabled(ctx context.Context, userID uuid.UUID) (bool, error)
	SetEmailTwoFactorEnabled(ctx context.Context, userID uuid.UUID, enabled bool) error
}

// IsEmailTwoFactorEnabled returns the current value of
// users.two_factor_email_enabled. Implements
// repository.TwoFactorUserFlagRepository on the existing
// UserRepository so wiring stays a single line in main.go.
//
// Returns user.ErrUserNotFound when the row does not exist so the app
// layer can map "user disappeared mid-request" to the same
// 401 session_invalid response /auth/me uses.
func (r *UserRepository) IsEmailTwoFactorEnabled(ctx context.Context, userID uuid.UUID) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var enabled bool
	err := QueryRow(ctx, r.db,
		`SELECT two_factor_email_enabled FROM users WHERE id = $1`,
		userID,
	).Scan(&enabled)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, user.ErrUserNotFound
		}
		return false, fmt.Errorf("two_factor_user_flag: read: %w", err)
	}
	return enabled, nil
}

// SetEmailTwoFactorEnabled flips the boolean. Bumps users.updated_at
// so admin tooling that filters "users updated in the last N days"
// correctly surfaces 2FA enrolment events.
func (r *UserRepository) SetEmailTwoFactorEnabled(ctx context.Context, userID uuid.UUID, enabled bool) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	res, err := r.db.ExecContext(ctx,
		`UPDATE users SET two_factor_email_enabled = $2, updated_at = NOW() WHERE id = $1`,
		userID, enabled,
	)
	if err != nil {
		return fmt.Errorf("two_factor_user_flag: write: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("two_factor_user_flag: rows affected: %w", err)
	}
	if rows == 0 {
		return user.ErrUserNotFound
	}
	return nil
}
