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

// sendAuthResponse checks X-Auth-Mode header to decide between
// session cookies (web) and token body (mobile).
func (h *AuthHandler) sendAuthResponse(w http.ResponseWriter, r *http.Request, status int, output *auth.AuthOutput) {
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

	// Mobile mode: return tokens in response body
	if r.Header.Get("X-Auth-Mode") == "token" {
		res.JSON(w, status, response.NewAuthResponseWithOrg(output.User, orgCtx, output.AccessToken, output.RefreshToken))
		return
	}

	// Web mode: create session, set cookies, return user only.
	// The session carries the fully-resolved effective permission
	// set (static defaults + org overrides) so the RequirePermission
	// middleware can honor customized roles without a DB round-trip.
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
