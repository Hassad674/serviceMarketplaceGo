package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"marketplace-backend/internal/app/auth"
	"marketplace-backend/internal/domain/twofactor"
	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/handler/middleware"
	"marketplace-backend/internal/port/repository"
	res "marketplace-backend/pkg/response"
	"marketplace-backend/pkg/validator"
)

// TwoFactorEnabler is the narrow contract the enable/disable endpoints
// need from the postgres adapter. Defined locally so the handler does
// not import the wide UserRepository for two methods.
type TwoFactorEnabler interface {
	IsEmailTwoFactorEnabled(ctx context.Context, userID uuid.UUID) (bool, error)
	SetEmailTwoFactorEnabled(ctx context.Context, userID uuid.UUID, enabled bool) error
}

// TwoFactorChallenger is the narrow contract the enable endpoint needs
// to issue a confirmation code before flipping the flag, and the
// verify-2fa endpoint needs to complete login. Implemented by the
// twofactor app service in main.go.
type TwoFactorChallenger interface {
	RequestChallenge(ctx context.Context, in auth.TwoFactorChallengeRequest) (uuid.UUID, error)
	VerifyChallenge(ctx context.Context, userID uuid.UUID, code string) error
}

// TwoFactorPasswordVerifier is the narrow contract the disable endpoint
// needs to confirm the caller's current password before turning 2FA
// off. We don't reuse the auth service's full ChangePassword because
// the disable flow only needs a "verify, don't rotate" check.
type TwoFactorPasswordVerifier interface {
	VerifyPassword(ctx context.Context, userID uuid.UUID, password string) error
}

// AttachTwoFactor wires the B.6 dependencies onto the auth handler.
// Returning the receiver keeps main.go fluent. All three deps are
// optional — passing nil disables the corresponding endpoint group at
// call time (handler returns 404 / 503 rather than panicking).
func (h *AuthHandler) AttachTwoFactor(
	flag TwoFactorEnabler,
	challenger TwoFactorChallenger,
	passwordVerifier TwoFactorPasswordVerifier,
) *AuthHandler {
	h.twoFactorFlag = flag
	h.twoFactorChallenger = challenger
	h.twoFactorPasswords = passwordVerifier
	return h
}

// VerifyTwoFactor completes a login that was gated by the 2FA flag.
// Body shape: { user_id, code }. On success the response body matches
// the regular Login response (token mode emits the bearer pair, web
// mode the session cookie). Mismatched / expired / exhausted codes
// map to user-facing errors via the same handleTwoFactorError helper.
func (h *AuthHandler) VerifyTwoFactor(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID string `json:"user_id"`
		Code   string `json:"code"`
	}
	if err := validator.DecodeJSON(r, &req); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if errs := validator.ValidateRequired(map[string]string{
		"user_id": req.UserID,
		"code":    req.Code,
	}); errs != nil {
		res.ValidationError(w, errs)
		return
	}
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_request", "user_id must be a valid uuid")
		return
	}

	output, err := h.authService.CompleteLoginWithTwoFactor(
		r.Context(), userID, req.Code, h.sessionFingerprint(r),
	)
	if err != nil {
		handleTwoFactorError(w, err)
		return
	}

	h.sendAuthResponse(w, r, http.StatusOK, output)
}

// EnableTwoFactor opts the authenticated user into email 2FA. The flow
// is two-step:
//  1. First call (no body OR body without code): issue a confirmation
//     challenge to the user's email and return 202 + challenge_id.
//  2. Second call ({code}): verify the code; on success flip the flag
//     to true and return 200 + {enabled: true}.
//
// Splitting it this way means the user proves they can read their own
// inbox before 2FA is on — otherwise a typo in their email address
// would lock them out of their own account.
func (h *AuthHandler) EnableTwoFactor(w http.ResponseWriter, r *http.Request) {
	if h.twoFactorFlag == nil || h.twoFactorChallenger == nil {
		res.Error(w, http.StatusServiceUnavailable, "feature_unavailable", "two-factor is not configured")
		return
	}
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	var req struct {
		Code string `json:"code"`
	}
	// Empty body is allowed — interpreted as "issue confirmation
	// challenge". Only surface decode errors when the body is non-empty.
	_ = validator.DecodeJSON(r, &req)

	if req.Code == "" {
		u, err := h.authService.GetMe(r.Context(), userID)
		if err != nil {
			handleAuthError(w, err)
			return
		}
		challengeID, err := h.twoFactorChallenger.RequestChallenge(r.Context(), auth.TwoFactorChallengeRequest{
			UserID:        userID,
			EmailTo:       u.Email,
			ClientIP:      h.sessionFingerprint(r).IPAnonymized,
			UserAgentHash: h.sessionFingerprint(r).UserAgentHash,
		})
		if err != nil {
			slog.Error("two_factor: enable issue challenge failed", "user_id", userID, "error", err)
			res.Error(w, http.StatusInternalServerError, "internal_error", "failed to issue confirmation code")
			return
		}
		res.JSON(w, http.StatusAccepted, map[string]any{
			"requires_confirmation": true,
			"challenge_id":          challengeID.String(),
		})
		return
	}

	if err := h.twoFactorChallenger.VerifyChallenge(r.Context(), userID, req.Code); err != nil {
		handleTwoFactorError(w, err)
		return
	}
	if err := h.twoFactorFlag.SetEmailTwoFactorEnabled(r.Context(), userID, true); err != nil {
		slog.Error("two_factor: flip flag on failed", "user_id", userID, "error", err)
		res.Error(w, http.StatusInternalServerError, "internal_error", "failed to enable two-factor")
		return
	}
	res.JSON(w, http.StatusOK, map[string]any{"enabled": true})
}

// DisableTwoFactor opts the authenticated user out of email 2FA. The
// caller must include their CURRENT password in the body to prove
// they're not a bystander hijacking an unlocked browser. On success
// the flag flips to false and the next login skips the 2FA gate.
func (h *AuthHandler) DisableTwoFactor(w http.ResponseWriter, r *http.Request) {
	if h.twoFactorFlag == nil || h.twoFactorPasswords == nil {
		res.Error(w, http.StatusServiceUnavailable, "feature_unavailable", "two-factor is not configured")
		return
	}
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}
	var req struct {
		CurrentPassword string `json:"current_password"`
	}
	if err := validator.DecodeJSON(r, &req); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if errs := validator.ValidateRequired(map[string]string{
		"current_password": req.CurrentPassword,
	}); errs != nil {
		res.ValidationError(w, errs)
		return
	}
	if err := h.twoFactorPasswords.VerifyPassword(r.Context(), userID, req.CurrentPassword); err != nil {
		if errors.Is(err, user.ErrInvalidCredentials) {
			res.Error(w, http.StatusUnauthorized, "invalid_credentials", "current password is incorrect")
			return
		}
		handleAuthError(w, err)
		return
	}
	if err := h.twoFactorFlag.SetEmailTwoFactorEnabled(r.Context(), userID, false); err != nil {
		slog.Error("two_factor: flip flag off failed", "user_id", userID, "error", err)
		res.Error(w, http.StatusInternalServerError, "internal_error", "failed to disable two-factor")
		return
	}
	res.JSON(w, http.StatusOK, map[string]any{"enabled": false})
}

// handleTwoFactorError maps the twofactor + repository sentinels to
// HTTP responses. Kept separate from handleAuthError because the
// 2FA surface has its own codes (invalid_code, challenge_expired,
// too_many_attempts, no_challenge) that don't make sense outside the
// 2FA flow.
func handleTwoFactorError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, twofactor.ErrChallengeNotFound),
		errors.Is(err, repository.ErrTwoFactorChallengeNotFound):
		res.Error(w, http.StatusBadRequest, "no_challenge", "no pending verification code — request a new one")
	case errors.Is(err, twofactor.ErrChallengeExpired):
		res.Error(w, http.StatusBadRequest, "challenge_expired", "the verification code has expired — request a new one")
	case errors.Is(err, twofactor.ErrAttemptsExhausted):
		res.Error(w, http.StatusTooManyRequests, "too_many_attempts", "too many incorrect attempts — request a new code")
	case errors.Is(err, twofactor.ErrCodeMismatch):
		res.Error(w, http.StatusBadRequest, "invalid_code", "the verification code is incorrect")
	case errors.Is(err, twofactor.ErrUserIDRequired):
		res.Error(w, http.StatusBadRequest, "invalid_request", "user_id is required")
	case errors.Is(err, user.ErrUserNotFound):
		res.Error(w, http.StatusUnauthorized, "session_invalid", "session is no longer valid — please sign in again")
	default:
		slog.Error("unhandled two-factor error", "error", err.Error())
		res.Error(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
	}
}
