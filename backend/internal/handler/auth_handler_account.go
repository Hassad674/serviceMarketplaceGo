package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"marketplace-backend/internal/app/auth"
	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/handler/middleware"
	res "marketplace-backend/pkg/response"
	"marketplace-backend/pkg/validator"
)

// changeEmailRequest is the body of POST /api/v1/auth/change-email.
// CurrentPassword caps at 128 to match the LoginRequest contract;
// NewEmail caps at 254 (RFC 5321). Both are required — the handler
// rejects the request with `invalid_request` before reaching the app
// layer when either is missing.
type changeEmailRequest struct {
	CurrentPassword string `json:"current_password"`
	NewEmail        string `json:"new_email"`
}

// changePasswordRequest is the body of POST /api/v1/auth/change-password.
// Both fields are required. The new_password length cap (128) is the
// same as RegisterRequest so the validator-bounded surface stays
// uniform across credential-mutating endpoints.
type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// ChangeEmail handles POST /api/v1/auth/change-email. The route is
// mounted behind the auth middleware so the user_id always comes
// from the JWT context — never from the request body.
//
// Wire shape on success: 200 OK with `{"data": {"email": "<new>"},
// "meta": {"request_id": "..."}}`. The session_version is bumped
// server-side and every Redis session for the user is purged, so the
// caller's existing access token will fail the next middleware
// version check and they will be forced to log in again.
func (h *AuthHandler) ChangeEmail(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	var req changeEmailRequest
	if err := validator.DecodeJSON(r, &req); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	if errs := validator.ValidateRequired(map[string]string{
		"current_password": req.CurrentPassword,
		"new_email":        req.NewEmail,
	}); errs != nil {
		res.ValidationError(w, errs)
		return
	}

	updated, err := h.authService.ChangeEmail(r.Context(), auth.ChangeEmailInput{
		UserID:          userID,
		CurrentPassword: req.CurrentPassword,
		NewEmail:        req.NewEmail,
	})
	if err != nil {
		handleChangeAccountError(w, err)
		return
	}

	res.JSON(w, http.StatusOK, envelope(r, map[string]any{
		"email": updated.Email,
	}))
}

// ChangePassword handles POST /api/v1/auth/change-password. Mirrors
// ChangeEmail's shape: auth-middleware-protected, JSON body, 200 OK
// envelope on success. The response payload is intentionally minimal
// — `{"ok": true}` — because there is nothing to echo back to the
// caller (a password is never returned).
func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	var req changePasswordRequest
	if err := validator.DecodeJSON(r, &req); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	if errs := validator.ValidateRequired(map[string]string{
		"current_password": req.CurrentPassword,
		"new_password":     req.NewPassword,
	}); errs != nil {
		res.ValidationError(w, errs)
		return
	}

	if err := h.authService.ChangePassword(r.Context(), auth.ChangePasswordInput{
		UserID:          userID,
		CurrentPassword: req.CurrentPassword,
		NewPassword:     req.NewPassword,
	}); err != nil {
		handleChangeAccountError(w, err)
		return
	}

	res.JSON(w, http.StatusOK, envelope(r, map[string]any{
		"ok": true,
	}))
}

// envelope wraps a payload in the project's `{data, meta}` response
// shape. Kept local to the change-email/change-password endpoints —
// the legacy auth flows (login/register/refresh) are intentionally
// not migrated in this round (out of scope, would break frontends).
func envelope(r *http.Request, data any) map[string]any {
	return map[string]any{
		"data": data,
		"meta": map[string]any{
			"request_id": middleware.GetRequestID(r.Context()),
		},
	}
}

// handleChangeAccountError maps the typed errors emitted by
// ChangeEmail / ChangePassword to HTTP responses. It is separate from
// handleAuthError because the credential-rotation surface has its own
// codes (`same_email`, `same_password`) and a different mapping for
// ErrInvalidCredentials (401 here, matching the brief — the wrong
// password on a self-service rotation must NOT be a 403 because the
// caller is already authenticated, the password just doesn't match).
func handleChangeAccountError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, user.ErrInvalidEmail):
		res.Error(w, http.StatusBadRequest, "invalid_email", err.Error())
	case errors.Is(err, user.ErrWeakPassword):
		res.Error(w, http.StatusBadRequest, "weak_password", err.Error())
	case errors.Is(err, user.ErrSameEmail):
		res.Error(w, http.StatusBadRequest, "same_email", err.Error())
	case errors.Is(err, user.ErrSamePassword):
		res.Error(w, http.StatusBadRequest, "same_password", err.Error())
	case errors.Is(err, user.ErrInvalidCredentials):
		res.Error(w, http.StatusUnauthorized, "invalid_credentials", err.Error())
	case errors.Is(err, user.ErrEmailAlreadyExists):
		res.Error(w, http.StatusConflict, "email_already_exists", err.Error())
	case errors.Is(err, user.ErrUserNotFound):
		// A self-service call that cannot find the caller's row is
		// the "session is no longer valid" signal — the JWT is still
		// cryptographically valid, but the underlying account was
		// deleted between issuance and now. Map to 401 so the client
		// clears state and redirects to /login (matching the /me
		// handler convention).
		res.Error(w, http.StatusUnauthorized, "session_invalid", "session is no longer valid — please sign in again")
	case errors.Is(err, user.ErrUnauthorized):
		res.Error(w, http.StatusUnauthorized, "unauthorized", err.Error())
	default:
		slog.Error("unhandled change-account error", "error", err.Error())
		res.Error(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
	}
}
