package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/audit"
	"marketplace-backend/internal/domain/user"
)

// ChangeEmailInput is the typed input for Service.ChangeEmail. Wraps
// the three values so adding an optional field later (e.g. a metadata
// hint for audit) does not break the constructor signature.
type ChangeEmailInput struct {
	UserID          uuid.UUID
	CurrentPassword string
	NewEmail        string
}

// ChangePasswordInput is the typed input for Service.ChangePassword.
type ChangePasswordInput struct {
	UserID          uuid.UUID
	CurrentPassword string
	NewPassword     string
}

// ChangeEmail updates the authenticated user's email address after
// verifying the current password. The new address is normalised by
// the Email value object (trim + lowercase) and rejected if either
//   - it does not parse,
//   - it is identical to the user's current address (case-insensitive),
//   - or another account already owns it.
//
// The session is invalidated on success (BumpSessionVersion + delete
// every Redis session row) so a stolen access token presented after
// the change immediately fails the middleware version check. This
// mirrors the ResetPassword kill-switch policy — an email change is a
// security-sensitive credential mutation and must take effect
// everywhere immediately.
//
// Audit row written on success: action `auth.change_email`, metadata
// `{old_email, new_email}`. Failures are NOT audited at this layer —
// the handler-level brute-force / rate-limit gates own that surface.
func (s *Service) ChangeEmail(ctx context.Context, in ChangeEmailInput) (*user.User, error) {
	newEmail, err := user.NewEmail(in.NewEmail)
	if err != nil {
		return nil, err
	}

	u, err := s.users.GetByID(ctx, in.UserID)
	if err != nil {
		return nil, fmt.Errorf("change email: get user: %w", err)
	}

	// Defence-in-depth: refuse the call when the account is in a state
	// that should not be self-mutating its credentials. The middleware
	// already blocks suspended/banned tokens at the perimeter, but a
	// freshly-soft-deleted user could still have a valid access token
	// — the kill switch must take precedence over self-service.
	if u.IsBanned() || u.IsSuspended() || u.IsScheduledForDeletion() {
		return nil, user.ErrUnauthorized
	}

	if err := s.hasher.Compare(u.HashedPassword, in.CurrentPassword); err != nil {
		return nil, user.ErrInvalidCredentials
	}

	// NewEmail() already lowercases + trims, so a direct compare is
	// the canonical check.
	if newEmail.String() == strings.ToLower(strings.TrimSpace(u.Email)) {
		return nil, user.ErrSameEmail
	}

	exists, err := s.users.ExistsByEmail(ctx, newEmail.String())
	if err != nil {
		return nil, fmt.Errorf("change email: check uniqueness: %w", err)
	}
	if exists {
		return nil, user.ErrEmailAlreadyExists
	}

	oldEmail := u.Email
	u.Email = newEmail.String()
	u.UpdatedAt = time.Now()

	if err := s.users.Update(ctx, u); err != nil {
		// The repository surfaces ErrEmailAlreadyExists when the unique
		// index trips concurrently between ExistsByEmail and Update —
		// pass it through unchanged so the handler maps it to 409.
		if errors.Is(err, user.ErrEmailAlreadyExists) {
			return nil, user.ErrEmailAlreadyExists
		}
		return nil, fmt.Errorf("change email: update user: %w", err)
	}

	s.invalidateUserSessions(ctx, u.ID, "change_email")

	s.logAudit(ctx, audit.NewEntryInput{
		UserID:       &u.ID,
		Action:       audit.ActionChangeEmail,
		ResourceType: audit.ResourceTypeUser,
		ResourceID:   &u.ID,
		Metadata: map[string]any{
			"old_email": oldEmail,
			"new_email": u.Email,
		},
	})

	return u, nil
}

// ChangePassword rotates the authenticated user's password after
// verifying the current one. The new password is validated against
// the domain `NewPassword` rule (≥10 chars, upper + lower + digit +
// special). Reuse of the current password is rejected to prevent a
// "rotate to self" no-op.
//
// On success the user's session_version is bumped and every Redis
// session row for the user is deleted, mirroring the ResetPassword
// kill-switch. The access token already in the caller's hand
// continues to work until the next request hits a route guarded by
// the auth middleware — at that point the version check fails and
// the client must re-authenticate.
//
// Audit row written on success: action `auth.change_password`, NO
// password material in metadata.
func (s *Service) ChangePassword(ctx context.Context, in ChangePasswordInput) error {
	if _, err := user.NewPassword(in.NewPassword); err != nil {
		return err
	}

	u, err := s.users.GetByID(ctx, in.UserID)
	if err != nil {
		return fmt.Errorf("change password: get user: %w", err)
	}

	if u.IsBanned() || u.IsSuspended() || u.IsScheduledForDeletion() {
		return user.ErrUnauthorized
	}

	if err := s.hasher.Compare(u.HashedPassword, in.CurrentPassword); err != nil {
		return user.ErrInvalidCredentials
	}

	// Reject "new = current". Because the stored value is a bcrypt
	// hash, the only safe way to compare is to run the hasher's
	// constant-time Compare against the same hash with the new
	// plaintext. A nil error means the new password collides with
	// the current one.
	if err := s.hasher.Compare(u.HashedPassword, in.NewPassword); err == nil {
		return user.ErrSamePassword
	}

	hashed, err := s.hasher.Hash(in.NewPassword)
	if err != nil {
		return fmt.Errorf("change password: hash: %w", err)
	}

	u.HashedPassword = hashed
	u.UpdatedAt = time.Now()

	if err := s.users.Update(ctx, u); err != nil {
		return fmt.Errorf("change password: update user: %w", err)
	}

	s.invalidateUserSessions(ctx, u.ID, "change_password")

	s.logAudit(ctx, audit.NewEntryInput{
		UserID:       &u.ID,
		Action:       audit.ActionChangePassword,
		ResourceType: audit.ResourceTypeUser,
		ResourceID:   &u.ID,
		Metadata:     map[string]any{},
	})

	return nil
}

// invalidateUserSessions is the shared kill-switch used by ChangeEmail
// and ChangePassword. It bumps users.session_version (every existing
// access token fails the next middleware version check) and purges
// every Redis session row for the user (the cookie path is also dead
// on the next request).
//
// Both calls are best-effort: logging at WARN on failure but never
// propagating, so a Redis blip cannot make a successful credential
// change look like a failure to the caller (which would put them in
// a worse state than partial invalidation).
func (s *Service) invalidateUserSessions(ctx context.Context, userID uuid.UUID, op string) {
	if _, err := s.users.BumpSessionVersion(ctx, userID); err != nil {
		slog.Warn("auth: bump session_version failed",
			"op", op, "user_id", userID, "error", err)
	}
	if s.sessionSvc != nil {
		if err := s.sessionSvc.DeleteByUserID(ctx, userID); err != nil {
			slog.Warn("auth: delete sessions failed",
				"op", op, "user_id", userID, "error", err)
		}
	}
}
