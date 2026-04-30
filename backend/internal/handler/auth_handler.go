package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"marketplace-backend/internal/app/auth"
	orgapp "marketplace-backend/internal/app/organization"
	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/handler/dto/response"
	"marketplace-backend/internal/handler/middleware"
	"marketplace-backend/internal/port/service"
	"marketplace-backend/pkg/validator"
	res "marketplace-backend/pkg/response"
)

type AuthHandler struct {
	authService     *auth.Service
	orgService      *orgapp.Service
	sessionSvc      service.SessionService
	cookie          *CookieConfig
	bruteForce      service.BruteForceService     // SEC-07: optional. nil disables brute-force protection.
	passwordReset   service.BruteForceService     // SEC-07: throttle password reset requests too. May be the same instance with a tighter policy.
}

func NewAuthHandler(authService *auth.Service, orgService *orgapp.Service, sessionSvc service.SessionService, cookie *CookieConfig) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		orgService:  orgService,
		sessionSvc:  sessionSvc,
		cookie:      cookie,
	}
}

// WithBruteForce wires the SEC-07 throttles. Returning the receiver
// keeps the call site fluent at main.go without forcing every test
// that constructs an AuthHandler to know about brute-force protection.
//
// loginGuard tracks /auth/login attempts (5 per 15min, 30min lockout).
// passwordReset tracks /auth/forgot-password + /auth/reset-password
// attempts (3 per hour). Pass the same instance for both when the
// caller wants identical policies.
func (h *AuthHandler) WithBruteForce(loginGuard, passwordReset service.BruteForceService) *AuthHandler {
	h.bruteForce = loginGuard
	h.passwordReset = passwordReset
	return h
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email       string `json:"email"`
		Password    string `json:"password"`
		FirstName   string `json:"first_name"`
		LastName    string `json:"last_name"`
		DisplayName string `json:"display_name"`
		Role        string `json:"role"`
	}

	if err := validator.DecodeJSON(r, &req); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	// Common required fields
	if errs := validator.ValidateRequired(map[string]string{
		"email":    req.Email,
		"password": req.Password,
		"role":     req.Role,
	}); errs != nil {
		res.ValidationError(w, errs)
		return
	}

	if err := validator.ValidateRole(req.Role); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_role", err.Error())
		return
	}

	// Role-specific validation
	switch req.Role {
	case "provider":
		if errs := validator.ValidateRequired(map[string]string{
			"first_name": req.FirstName,
			"last_name":  req.LastName,
		}); errs != nil {
			res.ValidationError(w, errs)
			return
		}
		req.DisplayName = strings.TrimSpace(req.FirstName) + " " + strings.TrimSpace(req.LastName)
	case "agency", "enterprise":
		if errs := validator.ValidateRequired(map[string]string{
			"display_name": req.DisplayName,
		}); errs != nil {
			res.ValidationError(w, errs)
			return
		}
		// For companies, first/last name are optional
		if req.FirstName == "" {
			req.FirstName = req.DisplayName
		}
		if req.LastName == "" {
			req.LastName = ""
		}
	}

	output, err := h.authService.Register(r.Context(), auth.RegisterInput{
		Email:       req.Email,
		Password:    req.Password,
		FirstName:   req.FirstName,
		LastName:    req.LastName,
		DisplayName: req.DisplayName,
		Role:        user.Role(req.Role),
	})
	if err != nil {
		handleAuthError(w, err)
		return
	}

	h.sendAuthResponse(w, r, http.StatusCreated, output)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := validator.DecodeJSON(r, &req); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	if errs := validator.ValidateRequired(map[string]string{
		"email":    req.Email,
		"password": req.Password,
	}); errs != nil {
		res.ValidationError(w, errs)
		return
	}

	// SEC-07: brute-force protection. Check the lockout BEFORE doing
	// any password validation so a locked account cannot be probed for
	// timing differences. A failure to reach Redis fails open (we
	// trust the on-success short-circuit + the SessionVersion check)
	// rather than block every login during a Redis blip.
	if h.bruteForce != nil {
		if locked, err := h.bruteForce.IsLocked(r.Context(), req.Email); err == nil && locked {
			h.tooManyAttempts(w, r, h.bruteForce, req.Email)
			return
		}
	}

	output, err := h.authService.Login(r.Context(), auth.LoginInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		// SEC-07: every failed login bumps the per-email counter.
		// Errors that are NOT credential failures (e.g. account
		// suspended) still count — a remote attacker should not get
		// a free pass simply because the target account is suspended.
		if h.bruteForce != nil {
			if recordErr := h.bruteForce.RecordFailure(r.Context(), req.Email); recordErr != nil {
				slog.Warn("brute force record_failure failed", "email", req.Email, "error", recordErr)
			}
		}
		handleAuthError(w, err)
		return
	}

	if h.bruteForce != nil {
		if recordErr := h.bruteForce.RecordSuccess(r.Context(), req.Email); recordErr != nil {
			slog.Warn("brute force record_success failed", "email", req.Email, "error", recordErr)
		}
	}

	h.sendAuthResponse(w, r, http.StatusOK, output)
}

// tooManyAttempts writes a 429 response with a Retry-After header
// derived from the lockout TTL. Used by Login + ForgotPassword +
// ResetPassword so the body shape stays identical across throttled
// flows.
func (h *AuthHandler) tooManyAttempts(w http.ResponseWriter, r *http.Request, guard service.BruteForceService, email string) {
	retry, err := guard.RetryAfter(r.Context(), email)
	if err != nil {
		slog.Warn("brute force retry_after failed", "email", email, "error", err)
	}
	if retry < time.Second {
		retry = time.Second
	}
	seconds := int(retry.Seconds())
	w.Header().Set("Retry-After", strconv.Itoa(seconds))
	res.Error(w, http.StatusTooManyRequests, "too_many_attempts",
		"Too many failed attempts. Please try again later.")
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie("session_id"); err == nil {
		_ = h.sessionSvc.Delete(r.Context(), cookie.Value)
	}
	h.cookie.ClearSession(w)

	// SEC-13: emit logout audit event when we know which user is
	// signing out. The handler is mounted behind the auth middleware
	// so userID is normally present in context — anonymous calls (no
	// auth context) skip the audit row but still receive a 200 to
	// preserve the legacy semantics.
	if userID, ok := middleware.GetUserID(r.Context()); ok {
		h.authService.Logout(r.Context(), userID)
	}

	// SEC-06: mobile clients post their refresh token here so the
	// backend can blacklist it immediately. Decoding failures and an
	// absent body are silently ignored — the session cookie was
	// already cleared above and the access token expires on its own
	// 15-minute timer. Returning 200 in every branch keeps the client
	// flow simple (logout never fails from the user's POV).
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := validator.DecodeJSON(r, &req); err == nil && req.RefreshToken != "" {
		h.authService.RevokeRefreshToken(r.Context(), req.RefreshToken)
	}

	res.JSON(w, http.StatusOK, map[string]string{"message": "logged out"})
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}

	if err := validator.DecodeJSON(r, &req); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	if req.RefreshToken == "" {
		res.Error(w, http.StatusBadRequest, "invalid_request", "refresh_token is required")
		return
	}

	output, err := h.authService.RefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		handleAuthError(w, err)
		return
	}

	res.JSON(w, http.StatusOK, response.NewAuthResponse(output.User, output.AccessToken, output.RefreshToken))
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	u, err := h.authService.GetMe(r.Context(), userID)
	if err != nil {
		// A missing user row on /auth/me means the authenticated caller's
		// account has been deleted between the time the session/token was
		// issued and now — typically when an operator leaves their org
		// and their account is hard-deleted (see R16). The JWT is still
		// cryptographically valid, but semantically the session is dead.
		// Return 401 session_invalid so the client clears state and
		// redirects to login, instead of 404 which frontends treat as a
		// benign "not found" and keep polling, leaving the user stuck in
		// a zombie "logged-in-but-deleted" state.
		if errors.Is(err, user.ErrUserNotFound) {
			res.Error(w, http.StatusUnauthorized, "session_invalid", "session is no longer valid — please sign in again")
			return
		}
		handleAuthError(w, err)
		return
	}

	// Resolve the user's organization context if the org service is wired.
	// Not having an org is not an error (Providers are expected to have none).
	var orgCtx *orgapp.Context
	if h.orgService != nil {
		resolved, resolveErr := h.orgService.ResolveContext(r.Context(), userID)
		if resolveErr != nil {
			slog.Warn("failed to resolve org context", "user_id", userID, "error", resolveErr)
		} else {
			orgCtx = resolved
		}
	}

	res.JSON(w, http.StatusOK, response.NewMeResponse(u, orgCtx))
}

// WebSession creates a fresh web session for the bearer-authenticated
// caller and returns its session_id so a mobile app can inject the
// matching cookie into an in-app WebView before opening pages that
// require auth (e.g. the embedded Premium subscribe flow). Without
// this bridge the WebView starts cookie-less and the Next.js
// middleware redirects the user to /login even though the Flutter
// app is already signed in.
//
// The endpoint mirrors the session creation logic at the tail of
// Login (sessionSvc.Create with the same payload) and returns
// `{ session_id, max_age_seconds }` so the caller can set the cookie
// with the same TTL the web flow uses. The session is stored in
// Redis with the standard expiry, indistinguishable from a
// browser-issued one — so the WebView gets the same RBAC + permission
// surface as a regular web tab.
//
// Security posture: the endpoint is mounted behind the same Bearer
// token middleware as /auth/me. A stolen bearer token already has
// equivalent privilege, so emitting an extra session does NOT
// broaden the attack surface — it just lets the bearer-authenticated
// caller open authenticated web pages in-app.
func (h *AuthHandler) WebSession(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	u, err := h.authService.GetMe(r.Context(), userID)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			res.Error(w, http.StatusUnauthorized, "session_invalid", "session is no longer valid — please sign in again")
			return
		}
		handleAuthError(w, err)
		return
	}

	// Resolve org context so the session carries the same permission
	// surface a regular web login would (RequirePermission middleware
	// reads them straight off the session). Solo users return no org
	// context which is fine — no permission keys to project.
	var orgCtx *orgapp.Context
	input := service.CreateSessionInput{
		UserID:         u.ID,
		Role:           u.Role.String(),
		IsAdmin:        u.IsAdmin,
		SessionVersion: u.SessionVersion,
	}
	if h.orgService != nil {
		resolved, resolveErr := h.orgService.ResolveContext(r.Context(), userID)
		if resolveErr == nil && resolved != nil && resolved.Organization != nil {
			orgCtx = resolved
			id := resolved.Organization.ID
			input.OrganizationID = &id
			if resolved.Member != nil {
				input.OrgRole = string(resolved.Member.Role)
			}
		}
	}
	input.Permissions = permissionKeysFromOrgContext(orgCtx)

	session, err := h.sessionSvc.Create(r.Context(), input)
	if err != nil {
		slog.Error("failed to create web bridge session", "user_id", userID, "error", err)
		res.Error(w, http.StatusInternalServerError, "internal_error", "failed to create session")
		return
	}

	res.JSON(w, http.StatusOK, map[string]any{
		"session_id":      session.ID,
		"max_age_seconds": h.cookie.MaxAge,
	})
}

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
