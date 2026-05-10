package auth

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/session"
)

// recordSession writes a server-side audit row for a freshly issued
// refresh token (B.4). The helper:
//   1. Validates the refresh token to extract its JTI + ExpiresAt
//      claims. We validate (rather than parsing unsafely) so the row
//      always corresponds to a token the system itself trusts.
//   2. Constructs a domain Session value from the supplied login
//      method, parent JTI, and request fingerprint.
//   3. Inserts the row through the repository.
//
// Failures are logged at WARN and swallowed — the audit trail is
// best-effort by design (matching the audit-log policy in CLAUDE.md).
// A missing repository OR an empty fingerprint means we skip the row
// and emit a slog.Warn so the missing piece is visible in
// production logs but the auth flow continues.
func (s *Service) recordSession(
	ctx context.Context,
	userID uuid.UUID,
	refreshToken string,
	parentJTI string,
	method session.LoginMethod,
	fp SessionFingerprint,
) {
	if s.userSessions == nil {
		return
	}
	if fp.UserAgentHash == "" || fp.IPAnonymized == "" {
		slog.Warn("auth: session audit skipped — missing fingerprint",
			"user_id", userID,
			"login_method", method)
		return
	}

	claims, err := s.tokens.ValidateRefreshToken(refreshToken)
	if err != nil {
		slog.Warn("auth: session audit skipped — refresh token validation failed",
			"user_id", userID,
			"error", err)
		return
	}
	if claims.JTI == "" {
		// Tokens minted by this codebase always carry a JTI (see
		// pkg/crypto/jwt.go), so a missing JTI here is a bug to log
		// loudly, not a recoverable case.
		slog.Warn("auth: session audit skipped — fresh refresh token has no JTI",
			"user_id", userID,
			"login_method", method)
		return
	}

	row, err := session.New(session.NewInput{
		UserID:        userID,
		JTI:           claims.JTI,
		ParentJTI:     parentJTI,
		UserAgentHash: fp.UserAgentHash,
		IPAnonymized:  fp.IPAnonymized,
		LoginMethod:   method,
		ExpiresAt:     claims.ExpiresAt,
	})
	if err != nil {
		slog.Warn("auth: session audit build failed",
			"user_id", userID,
			"login_method", method,
			"error", err)
		return
	}
	if err := s.userSessions.Create(ctx, row); err != nil {
		slog.Warn("auth: session audit insert failed",
			"user_id", userID,
			"login_method", method,
			"error", err)
	}
}

// revokeSessionByJTI marks the session attached to jti as revoked.
// Best-effort: a nil repo or a missing row is logged at WARN and
// otherwise silent so the caller's logout / refresh flow keeps
// progressing.
func (s *Service) revokeSessionByJTI(ctx context.Context, jti string) {
	if s.userSessions == nil || jti == "" {
		return
	}
	if err := s.userSessions.Revoke(ctx, jti); err != nil {
		slog.Warn("auth: session revoke failed",
			"jti", jti,
			"error", err)
	}
}

// revokeAllSessionsForUser kills every still-active session row
// attached to the user. Used on token-reuse detection (assume the
// account is compromised). Best-effort.
func (s *Service) revokeAllSessionsForUser(ctx context.Context, userID uuid.UUID) {
	if s.userSessions == nil || userID == uuid.Nil {
		return
	}
	if err := s.userSessions.RevokeAllForUser(ctx, userID); err != nil {
		slog.Warn("auth: session revoke_all failed",
			"user_id", userID,
			"error", err)
	}
}
