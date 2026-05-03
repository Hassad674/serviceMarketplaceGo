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
	res "marketplace-backend/pkg/response"
	"marketplace-backend/pkg/validator"
)

type AuthHandler struct {
	authService   *auth.Service
	orgService    *orgapp.Service
	sessionSvc    service.SessionService
	cookie        *CookieConfig
	bruteForce    service.BruteForceService // SEC-07: optional. nil disables brute-force protection.
	passwordReset service.BruteForceService // SEC-07: throttle password reset requests too. May be the same instance with a tighter policy.

	// F.5 S7: failClosedInProd controls how a Redis-side
	// brute-force-service error is handled. true (production) returns
	// 503 so the lockout cannot be bypassed via a Redis outage; false
	// (dev/test) preserves legacy fail-OPEN. Set via WithFailClosed.
	failClosedInProd bool
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

// WithFailClosed (F.5 S7) makes a Redis error in the brute-force
// IsLocked path return 503 to the client. Without this flag, an
// attacker could bypass the lockout by triggering a Redis blip — the
// handler's previous `err == nil && locked` pattern silently
// swallowed the error. Toggled on for production, off for dev.
func (h *AuthHandler) WithFailClosed(failClosedInProd bool) *AuthHandler {
	h.failClosedInProd = failClosedInProd
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

	// F.5 S5: anti-enumeration neutral response. The service returns
	// SilentDuplicate=true when the email is already registered. The
	// wire response is "202 Accepted with neutral message" — wire
	// shape indistinguishable from a fresh registration. A probe
	// cannot decide whether the email is taken via status code or
	// payload (the legitimate owner gets a signal email; the probe
	// learns nothing).
	if output != nil && output.SilentDuplicate {
		res.JSON(w, http.StatusAccepted, map[string]string{
			"message": "Registration request received — check your email for confirmation.",
		})
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

	// SEC-07 + F.5 S7: brute-force protection. Check the lockout
	// BEFORE doing any password validation so a locked account cannot
	// be probed for timing differences. Failure-mode policy is
	// environment-aware:
	//   * production : a Redis error returns 503 so the lockout cannot
	//     be bypassed via a Redis outage. Without this guard, the
	//     legacy `err == nil && locked` check silently swallowed the
	//     error and let an attacker keep trying credentials at full
	//     speed during the blip.
	//   * dev/test    : fail-OPEN preserved (slog.Error breadcrumb so
	//     the contributor sees their broken local Redis).
	if h.bruteForce != nil {
		locked, err := h.bruteForce.IsLocked(r.Context(), req.Email)
		switch {
		case err == nil && locked:
			h.tooManyAttempts(w, r, h.bruteForce, req.Email)
			return
		case err != nil && h.failClosedInProd:
			slog.Error("brute force IsLocked failed — failing closed",
				"email", req.Email, "error", err)
			res.Error(w, http.StatusServiceUnavailable,
				"auth_unavailable",
				"authentication backend is degraded — retry shortly")
			return
		case err != nil:
			slog.Error("brute force IsLocked failed — failing open in non-prod",
				"email", req.Email, "error", err)
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
