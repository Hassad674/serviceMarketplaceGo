package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	appmoderation "marketplace-backend/internal/app/moderation"
	"marketplace-backend/internal/domain/audit"
	"marketplace-backend/internal/domain/moderation"
	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/port/repository"
)

// resolveOrgContext returns the user's org context at login/refresh time.
// Returns nil when the user has no org or when the provisioner is not wired.
func (s *Service) resolveOrgContext(ctx context.Context, userID uuid.UUID) (*orgContext, error) {
	if s.orgs == nil {
		return nil, nil
	}
	orgCtx, err := s.orgs.ResolveContext(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("resolve org context: %w", err)
	}
	return orgCtx, nil
}

func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (*AuthOutput, error) {
	claims, err := s.tokens.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, user.ErrUnauthorized
	}

	// SEC-06: refresh-token rotation. Reject the request if the JTI is
	// already on the blacklist — that means the token has already been
	// rotated (legitimate user) or revoked (logout) and the caller is
	// presenting a stale or stolen credential. A blacklist read failure
	// fails open (we trust the SessionVersion + token signature checks
	// to catch a real compromise) so a Redis blip does not lock every
	// user out of the app.
	if s.refreshBlacklist != nil && claims.JTI != "" {
		blacklisted, err := s.refreshBlacklist.Has(ctx, claims.JTI)
		if err != nil {
			slog.Warn("refresh blacklist read failed", "jti", claims.JTI, "error", err)
		} else if blacklisted {
			s.recordTokenReuse(ctx, claims.UserID, claims.JTI)
			return nil, user.ErrUnauthorized
		}
	}

	u, err := s.users.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, user.ErrUnauthorized
	}

	if u.IsSuspended() {
		return nil, user.NewSuspendedError(u.SuspensionReason)
	}
	if u.IsBanned() {
		return nil, user.NewBannedError(u.BanReason)
	}

	orgCtx, err := s.resolveOrgContext(ctx, u.ID)
	if err != nil {
		return nil, err
	}

	newAccessToken, err := s.tokens.GenerateAccessToken(buildAccessInput(u, orgCtx))
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	newRefreshToken, err := s.tokens.GenerateRefreshToken(u.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// SEC-13: emit token_refresh audit event.
	s.logAudit(ctx, audit.NewEntryInput{
		UserID:       &u.ID,
		Action:       audit.ActionTokenRefresh,
		ResourceType: audit.ResourceTypeUser,
		ResourceID:   &u.ID,
		Metadata:     map[string]any{},
	})

	// SEC-06: blacklist the OLD refresh token AFTER generating the new
	// pair. We use the original token's remaining time-to-expire as
	// the blacklist TTL so the entry self-evicts once the token would
	// have failed validation anyway. A blacklist write failure is
	// logged but not propagated — the caller already has a working
	// new pair, and the next replay will be blocked by the
	// SessionVersion check on the next mutation.
	if s.refreshBlacklist != nil && claims.JTI != "" {
		ttl := time.Until(claims.ExpiresAt)
		if err := s.refreshBlacklist.Add(ctx, claims.JTI, ttl); err != nil {
			slog.Warn("refresh blacklist add failed", "jti", claims.JTI, "error", err)
		}
	}

	return buildAuthOutput(u, orgCtx, newAccessToken, newRefreshToken), nil
}

// Logout records a logout audit event for the given user. Used by the
// auth handler after it has invalidated the session — exposes a method
// rather than logging from the handler so the audit emission stays in
// the app layer with the rest of the auth audit calls.
func (s *Service) Logout(ctx context.Context, userID uuid.UUID) {
	s.logAudit(ctx, audit.NewEntryInput{
		UserID:       &userID,
		Action:       audit.ActionLogout,
		ResourceType: audit.ResourceTypeUser,
		ResourceID:   &userID,
		Metadata:     map[string]any{},
	})
}

// RevokeRefreshToken blacklists the supplied refresh token's JTI so any
// subsequent /auth/refresh call presenting the same token fails with
// 401 Unauthorized. Used by the logout handler to immediately invalidate
// the mobile-mode token pair (web mode invalidates the session cookie
// instead, but we still call this to belt-and-braces the case where the
// same caller has both a session and a JWT pair).
//
// An invalid token, a token without a JTI, or a not-yet-wired blacklist
// is a silent no-op — the caller's logout intent is honored at the
// session layer and we do not surface a 500 just because there is
// nothing to blacklist.
func (s *Service) RevokeRefreshToken(ctx context.Context, refreshToken string) {
	if s.refreshBlacklist == nil || refreshToken == "" {
		return
	}
	claims, err := s.tokens.ValidateRefreshToken(refreshToken)
	if err != nil || claims.JTI == "" {
		return
	}
	ttl := time.Until(claims.ExpiresAt)
	if err := s.refreshBlacklist.Add(ctx, claims.JTI, ttl); err != nil {
		slog.Warn("refresh blacklist revoke failed", "jti", claims.JTI, "error", err)
	}
}

// recordTokenReuse writes an auth.token_reuse_detected audit row. The
// best-effort policy applies: any failure is logged at WARN and
// swallowed. The fact that we got here at all means the request will
// be rejected with 401, so even a dropped audit row does not affect
// the user's experience — only the SOC investigation surface.
func (s *Service) recordTokenReuse(ctx context.Context, userID uuid.UUID, jti string) {
	if s.audits == nil {
		return
	}
	uid := userID
	entry, err := audit.NewEntry(audit.NewEntryInput{
		UserID:       &uid,
		Action:       audit.ActionTokenReuseDetected,
		ResourceType: audit.ResourceTypeUser,
		ResourceID:   &uid,
		Metadata: map[string]any{
			"jti": jti,
		},
	})
	if err != nil {
		slog.Warn("audit: build token_reuse_detected entry failed", "error", err)
		return
	}
	if err := s.audits.Log(ctx, entry); err != nil {
		slog.Warn("audit: insert token_reuse_detected failed", "error", err)
	}
}

func (s *Service) GetMe(ctx context.Context, userID uuid.UUID) (*user.User, error) {
	return s.users.GetByID(ctx, userID)
}

func (s *Service) EnableReferrer(ctx context.Context, userID uuid.UUID) (*user.User, error) {
	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("enable referrer: get user: %w", err)
	}

	if u.Role != user.RoleProvider {
		return nil, user.ErrInvalidRole
	}

	u.EnableReferrer()
	u.UpdatedAt = time.Now()

	if err := s.users.Update(ctx, u); err != nil {
		return nil, fmt.Errorf("enable referrer: update user: %w", err)
	}

	return u, nil
}

func (s *Service) ForgotPassword(ctx context.Context, input ForgotPasswordInput) error {
	email, err := user.NewEmail(input.Email)
	if err != nil {
		return nil // Don't reveal if email exists
	}

	u, err := s.users.GetByEmail(ctx, email.String())
	if err != nil {
		return nil // Don't reveal if email exists
	}

	token := uuid.New().String()
	pr := &repository.PasswordReset{
		ID:        uuid.New(),
		UserID:    u.ID,
		Token:     token,
		ExpiresAt: time.Now().Add(1 * time.Hour),
		CreatedAt: time.Now(),
	}

	if err := s.resets.Create(ctx, pr); err != nil {
		return fmt.Errorf("create reset token: %w", err)
	}

	resetURL := s.frontendURL + "/reset-password?token=" + token
	if err := s.email.SendPasswordReset(ctx, u.Email, resetURL); err != nil {
		return fmt.Errorf("send reset email: %w", err)
	}

	s.logAudit(ctx, audit.NewEntryInput{
		UserID:       &u.ID,
		Action:       audit.ActionPasswordResetRequest,
		ResourceType: audit.ResourceTypeUser,
		ResourceID:   &u.ID,
		Metadata: map[string]any{
			"email": u.Email,
		},
	})

	return nil
}

func (s *Service) ResetPassword(ctx context.Context, input ResetPasswordInput) error {
	if _, err := user.NewPassword(input.NewPassword); err != nil {
		return err
	}

	pr, err := s.resets.GetByToken(ctx, input.Token)
	if err != nil {
		return user.ErrUnauthorized
	}

	if pr.Used || pr.ExpiresAt.Before(time.Now()) {
		return user.ErrUnauthorized
	}

	hashedPassword, err := s.hasher.Hash(input.NewPassword)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	u, err := s.users.GetByID(ctx, pr.UserID)
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}

	u.HashedPassword = hashedPassword
	if err := s.users.Update(ctx, u); err != nil {
		return fmt.Errorf("update password: %w", err)
	}

	if err := s.resets.MarkUsed(ctx, pr.ID); err != nil {
		return fmt.Errorf("mark token used: %w", err)
	}

	// SEC-16: a successful reset is the user's "kill switch" — every
	// session that was alive before the reset must be invalidated.
	// Two complementary mechanisms are used:
	//   1. Bump session_version so any access token issued before the
	//      reset fails the middleware version check on its next request
	//      (mobile-friendly: no shared session storage required).
	//   2. Delete every session row in Redis so the cookie-based web
	//      session is gone immediately on the next request.
	// Failures are logged but do NOT fail the reset itself — the password
	// is already changed, refusing the call would put the user in a
	// worse state ("your password is changed but… error?").
	if _, err := s.users.BumpSessionVersion(ctx, u.ID); err != nil {
		slog.Warn("auth: reset_password bump session_version failed",
			"user_id", u.ID, "error", err)
	}
	if s.sessionSvc != nil {
		if err := s.sessionSvc.DeleteByUserID(ctx, u.ID); err != nil {
			slog.Warn("auth: reset_password delete sessions failed",
				"user_id", u.ID, "error", err)
		}
	}

	s.logAudit(ctx, audit.NewEntryInput{
		UserID:       &u.ID,
		Action:       audit.ActionPasswordResetComplete,
		ResourceType: audit.ResourceTypeUser,
		ResourceID:   &u.ID,
		Metadata: map[string]any{
			"email": u.Email,
		},
	})

	return nil
}

// notifyDuplicateRegistrationAttempt sends a best-effort "someone
// tried to register your account" signal email to the legitimate
// owner of the address, plus an audit row. Used by Register() (F.5 S5)
// when the email is already taken — instead of leaking that fact to
// the API caller via a 409, the service silently informs the real
// user and returns an indistinguishable "neutral success" output.
//
// Email + audit failures are logged at WARN and swallowed — a probe
// should not be able to detect a Redis blip from the response shape,
// and the security signal is best-effort by definition (the address
// is already in use, so the user can still sign in via the normal
// flow).
func (s *Service) notifyDuplicateRegistrationAttempt(ctx context.Context, emailAddr string) {
	// Audit row first — written regardless of email service availability,
	// so a SOC investigation can correlate registration probes even when
	// the email transport is degraded.
	if s.audits != nil {
		entry, err := audit.NewEntry(audit.NewEntryInput{
			Action:       audit.ActionLoginFailure,
			ResourceType: audit.ResourceTypeUser,
			Metadata: map[string]any{
				"email":  emailAddr,
				"reason": "register_duplicate_silent",
			},
		})
		if err != nil {
			slog.Warn("audit: build register_duplicate entry failed", "error", err)
		} else if logErr := s.audits.Log(ctx, entry); logErr != nil {
			slog.Warn("audit: insert register_duplicate failed", "error", logErr)
		}
	}

	if s.email == nil {
		return
	}
	subject := "Tentative d'inscription avec votre adresse email"
	html := "<p>Bonjour,</p>" +
		"<p>Quelqu'un vient d'essayer de créer un compte sur la plateforme avec votre adresse email. " +
		"Votre compte existant n'a pas été modifié.</p>" +
		"<p>Si vous êtes à l'origine de cette tentative, vous pouvez l'ignorer ou vous connecter " +
		"directement via la page de connexion.</p>" +
		"<p>Si ce n'est pas vous, nous vous recommandons de vérifier que votre mot de passe est " +
		"toujours sûr (option \"Mot de passe oublié\" sur la page de connexion).</p>"
	if err := s.email.SendNotification(ctx, emailAddr, subject, html); err != nil {
		slog.Warn("auth: send duplicate-registration notification failed",
			"error", err)
	}
}

// moderateDisplayName runs the synchronous blocking gate against the
// user's public-facing identity. Returns nil when the moderation
// orchestrator is not wired (test scenarios) or when the content
// passes; returns user.ErrDisplayNameInappropriate when the engine
// flags the input above the blocking threshold.
//
// The threshold is 0.50 — the strictest tier in the policy matrix —
// because a public profile name has zero legitimate use case for
// toxic terms. False positives are reversible by admin review on the
// next attempt; false negatives create a permanent SEO + harassment
// surface.
//
// Engine errors (OpenAI 5xx, network) are propagated as a generic
// failure so the user retries. We deliberately fail closed: a
// registration that we cannot moderate is a registration we refuse.
func (s *Service) moderateDisplayName(ctx context.Context, u *user.User) error {
	if s.moderationOrchestrator == nil {
		return nil
	}
	combined := strings.TrimSpace(strings.Join([]string{u.DisplayName, u.FirstName, u.LastName}, " "))
	if combined == "" {
		return nil
	}
	_, err := s.moderationOrchestrator.Moderate(ctx, appmoderation.ModerateInput{
		ContentType:       moderation.ContentTypeUserDisplayName,
		ContentID:         u.ID,
		AuthorUserID:      &u.ID,
		Text:              combined,
		BlockingMode:      true,
		BlockingThreshold: 0.50,
	})
	if errors.Is(err, moderation.ErrContentBlocked) {
		return user.ErrDisplayNameInappropriate
	}
	if err != nil {
		return fmt.Errorf("moderate display name: %w", err)
	}
	return nil
}
