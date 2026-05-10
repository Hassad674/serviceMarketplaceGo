package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"marketplace-backend/internal/app/auth"
	orgapp "marketplace-backend/internal/app/organization"
	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/handler/dto/response"
	"marketplace-backend/internal/handler/middleware"
	"marketplace-backend/internal/port/service"
	res "marketplace-backend/pkg/response"
	"marketplace-backend/pkg/validator"
)

// WSToken issues a short-lived single-use token for WebSocket authentication.
// The frontend calls this via the same-origin proxy (httpOnly session cookie is
// sent automatically), then passes the token as a query param when connecting to
// the WebSocket on Railway directly. This avoids exposing the session_id.
func (h *AuthHandler) WSToken(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	token, err := h.sessionSvc.CreateWSToken(r.Context(), userID)
	if err != nil {
		res.Error(w, http.StatusInternalServerError, "internal_error", "failed to create ws token")
		return
	}

	res.JSON(w, http.StatusOK, map[string]string{"token": token})
}

func (h *AuthHandler) EnableReferrer(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	u, err := h.authService.EnableReferrer(r.Context(), userID)
	if err != nil {
		handleAuthError(w, err)
		return
	}

	res.JSON(w, http.StatusOK, response.NewUserResponse(u))
}

func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	if err := validator.DecodeJSON(r, &req); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	// SEC-07: throttle password-reset requests at 3 per hour per email
	// so an attacker cannot flood inboxes or brute-force the reset
	// token endpoint. We check the lockout BEFORE doing any work so
	// a locked email never sends a fresh reset email.
	if h.passwordReset != nil {
		if locked, err := h.passwordReset.IsLocked(r.Context(), req.Email); err == nil && locked {
			h.tooManyAttempts(w, r, h.passwordReset, req.Email)
			return
		}
		// Always record a failure (= attempt) — we cannot tell whether
		// the email exists without leaking that information, so every
		// request counts toward the cap regardless of outcome.
		if recordErr := h.passwordReset.RecordFailure(r.Context(), req.Email); recordErr != nil {
			slog.Warn("brute force forgot_password record_failure failed", "email", req.Email, "error", recordErr)
		}
	}

	// Always return 200 OK regardless of whether email exists (security)
	_ = h.authService.ForgotPassword(r.Context(), auth.ForgotPasswordInput{Email: req.Email})

	res.JSON(w, http.StatusOK, map[string]string{
		"message": "Si cette adresse existe, un email de réinitialisation a été envoyé.",
	})
}

func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token       string `json:"token"`
		NewPassword string `json:"new_password"`
	}
	if err := validator.DecodeJSON(r, &req); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	if errs := validator.ValidateRequired(map[string]string{
		"token":        req.Token,
		"new_password": req.NewPassword,
	}); errs != nil {
		res.ValidationError(w, errs)
		return
	}

	// SEC-07: throttle reset-token consumption per token so a stolen
	// token cannot be brute-forced through password attempts. We use
	// the token as the throttle key (instead of the email) because
	// the email is not in the request body — and tokens are
	// single-use anyway, so the cap protects against
	// password-guessing during the brief window the token is valid.
	if h.passwordReset != nil {
		if locked, err := h.passwordReset.IsLocked(r.Context(), req.Token); err == nil && locked {
			h.tooManyAttempts(w, r, h.passwordReset, req.Token)
			return
		}
	}

	err := h.authService.ResetPassword(r.Context(), auth.ResetPasswordInput{
		Token:       req.Token,
		NewPassword: req.NewPassword,
	})
	if err != nil {
		// Bump the per-token counter on failure so a guessing attack
		// against the new_password field (e.g. trying common
		// passwords) hits the cap.
		if h.passwordReset != nil {
			if recordErr := h.passwordReset.RecordFailure(r.Context(), req.Token); recordErr != nil {
				slog.Warn("brute force reset_password record_failure failed", "error", recordErr)
			}
		}
		handleAuthError(w, err)
		return
	}

	if h.passwordReset != nil {
		if recordErr := h.passwordReset.RecordSuccess(r.Context(), req.Token); recordErr != nil {
			slog.Warn("brute force reset_password record_success failed", "error", recordErr)
		}
	}

	res.JSON(w, http.StatusOK, map[string]string{
		"message": "Mot de passe réinitialisé avec succès.",
	})
}

// sendAuthResponse uses the X-Auth-Mode header to decide what shape the
// response BODY takes — but it ALWAYS creates a session and sets the
// httpOnly session cookie. The cookie is the long-lived "you're logged
// in" record; the bearer token (when returned in token mode) is a
// short-lived convenience for in-memory clients (mobile + admin SPA).
//
// Why the cookie is also set in token mode (admin SPA bug fix):
//
// The admin SPA stores the bearer in memory only (SEC-FINAL-07) — a
// hard reload drops it. To recover the session without forcing the
// user to log in again, the SPA's `AuthProvider` probes
// `GET /auth/me` with `credentials: "include"` on boot. That probe
// only succeeds when the backend issued a session cookie at login.
//
// Before this fix, the token-mode branch RETURNED EARLY before any
// session was created, so the SPA never received a Set-Cookie header
// and every reload booted into a logged-out state. The legacy
// "no cookie when token mode" behaviour had no security benefit
// (Dio on mobile never reads Set-Cookie anyway, and the cookie is
// httpOnly + Secure-in-prod so it cannot be exfiltrated by JS) and a
// concrete cost: every admin reload kicked the user back to /login.
func (h *AuthHandler) sendAuthResponse(w http.ResponseWriter, r *http.Request, status int, output *auth.AuthOutput) {
	// B.6: when 2FA is required the auth service skipped token issuance
	// — the response shape is intentionally narrow ({requires_2fa,
	// user_id, challenge_id}) and no session cookie is set. The client
	// is expected to prompt for the 6-digit code and call
	// /auth/login/verify-2fa to complete the login.
	if output != nil && output.RequiresTwoFactor {
		res.JSON(w, http.StatusOK, map[string]any{
			"requires_2fa": true,
			"user_id":      output.TwoFactorUserID.String(),
			"challenge_id": output.TwoFactorChallengeID.String(),
		})
		return
	}

	// Resolve the freshly created/loaded org context for inclusion in the
	// response payload. We re-query the org service rather than storing
	// the Context on AuthOutput to keep the auth package from leaking
	// internal/app/organization types outward.
	var orgCtx *orgapp.Context
	if h.orgService != nil {
		resolved, err := h.orgService.ResolveContext(r.Context(), output.User.ID)
		if err == nil {
			orgCtx = resolved
		}
	}

	// Always create the session + set cookies. The session carries the
	// fully-resolved effective permission set (static defaults + org
	// overrides) so the RequirePermission middleware can honor
	// customized roles without a DB round-trip.
	session, err := h.sessionSvc.Create(r.Context(), service.CreateSessionInput{
		UserID:         output.User.ID,
		Role:           output.User.Role.String(),
		IsAdmin:        output.User.IsAdmin,
		OrganizationID: output.OrganizationID,
		OrgRole:        output.OrgRole,
		Permissions:    permissionKeysFromOrgContext(orgCtx),
		SessionVersion: output.User.SessionVersion,
	})
	if err != nil {
		slog.Error("failed to create session", "error", err)
		res.Error(w, http.StatusInternalServerError, "internal_error", "failed to create session")
		return
	}
	h.cookie.SetSession(w, session.ID, output.User.Role.String())

	// Token mode: the response body carries the bearer (mobile +
	// admin SPA store it in memory). The session cookie set above is
	// the reload-recovery anchor for the admin SPA — mobile ignores
	// the Set-Cookie header (Dio has no CookieJar wired) and uses the
	// bearer exclusively, so the extra cookie is harmless.
	if r.Header.Get("X-Auth-Mode") == "token" {
		res.JSON(w, status, response.NewAuthResponseWithOrg(output.User, orgCtx, output.AccessToken, output.RefreshToken))
		return
	}

	// Web mode: response body omits the tokens — the cookies above are
	// the only credential the browser will replay on subsequent calls.
	res.JSON(w, status, response.NewMeResponse(output.User, orgCtx))
}

// permissionKeysFromOrgContext projects an org context's resolved
// permission set into the []string shape expected by
// CreateSessionInput and AccessTokenInput. Returns nil when the org
// context is missing (solo user) so the session omits the field.
func permissionKeysFromOrgContext(ctx *orgapp.Context) []string {
	if ctx == nil || len(ctx.Permissions) == 0 {
		return nil
	}
	out := make([]string, 0, len(ctx.Permissions))
	for _, p := range ctx.Permissions {
		out = append(out, string(p))
	}
	return out
}

func handleAuthError(w http.ResponseWriter, err error) {
	// Check for suspension/ban errors first — they carry a reason payload.
	var statusErr *user.AccountStatusError
	if errors.As(err, &statusErr) {
		code := "account_suspended"
		message := "Votre compte a \u00e9t\u00e9 suspendu"
		httpStatus := http.StatusForbidden
		if errors.Is(statusErr.Sentinel, user.ErrAccountBanned) {
			code = "account_banned"
			message = "Votre compte a \u00e9t\u00e9 banni"
		} else if errors.Is(statusErr.Sentinel, user.ErrAccountScheduledForDeletion) {
			// P5 (GDPR): soft-deleted account. 410 Gone tells the
			// frontend the resource is scheduled for deletion;
			// `reason` is the RFC3339 deleted_at timestamp so the
			// UI can compute the 30-day countdown without a
			// separate fetch.
			code = "account_scheduled_for_deletion"
			message = "Votre compte est planifi\u00e9 pour suppression"
			httpStatus = http.StatusGone
		}
		res.JSON(w, httpStatus, map[string]string{
			"error":   code,
			"message": message,
			"reason":  statusErr.Reason,
		})
		return
	}

	switch {
	case errors.Is(err, user.ErrInvalidEmail):
		res.Error(w, http.StatusBadRequest, "invalid_email", err.Error())
	case errors.Is(err, user.ErrWeakPassword):
		res.Error(w, http.StatusBadRequest, "weak_password", err.Error())
	case errors.Is(err, user.ErrEmailAlreadyExists):
		res.Error(w, http.StatusConflict, "email_exists", err.Error())
	case errors.Is(err, user.ErrDisplayNameInappropriate):
		res.Error(w, http.StatusUnprocessableEntity, "display_name_inappropriate",
			"This name violates our content guidelines. Please choose a different one.")
	case errors.Is(err, user.ErrInvalidCredentials):
		res.Error(w, http.StatusUnauthorized, "invalid_credentials", err.Error())
	case errors.Is(err, user.ErrUserNotFound):
		res.Error(w, http.StatusNotFound, "user_not_found", err.Error())
	case errors.Is(err, user.ErrUnauthorized):
		res.Error(w, http.StatusUnauthorized, "unauthorized", err.Error())
	case errors.Is(err, user.ErrInvalidRole):
		res.Error(w, http.StatusBadRequest, "invalid_role", err.Error())
	default:
		slog.Error("unhandled auth error", "error", err.Error())
		res.Error(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
	}
}
