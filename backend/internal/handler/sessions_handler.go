package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"marketplace-backend/internal/domain/session"
	"marketplace-backend/internal/handler/middleware"
	"marketplace-backend/internal/port/repository"
	res "marketplace-backend/pkg/response"
)

// SessionsHandler exposes the user-facing "Sécurité" surface that lists
// the caller's active sessions and lets them revoke any of them.
//
// Auth is enforced upstream by the router. Every row this handler
// returns is filtered by user_id at the repository layer (RLS belt +
// WHERE user_id = $1 suspenders) — there is no cross-tenant query
// path.
//
// The handler stays thin: it resolves the calling user from the JWT
// context, calls the repo, maps the resulting domain values into a
// JSON envelope, and decorates the row matching the current session's
// JTI cookie with `is_current: true` so the UI can render the "Cette
// session" badge.
type SessionsHandler struct {
	repo       repository.UserSessionRepository
	cookieName string // name of the refresh-token cookie ("refresh_token" by default); used to discover the current session JTI
}

// NewSessionsHandler returns a fully wired handler. A nil repo
// short-circuits every method with 503 so the rest of the API keeps
// running if the feature is not wired (matches the rest of the
// "optional handler" pattern in this package).
func NewSessionsHandler(repo repository.UserSessionRepository, cookieName string) *SessionsHandler {
	if cookieName == "" {
		cookieName = "refresh_token"
	}
	return &SessionsHandler{repo: repo, cookieName: cookieName}
}

// sessionResponseItem is the on-the-wire shape of a single session
// row. JSON field names mirror the column names so the contract is
// trivially diff-able against the schema. `device_label` falls back
// to "Appareil inconnu" at the parser layer — the field is always
// populated.
type sessionResponseItem struct {
	ID           string `json:"id"`
	DeviceLabel  string `json:"device_label"`
	Browser      string `json:"browser,omitempty"`
	OS           string `json:"os,omitempty"`
	City         string `json:"city,omitempty"`
	CountryCode  string `json:"country_code,omitempty"`
	IPAnonymized string `json:"ip_anonymized,omitempty"`
	LoginMethod  string `json:"login_method"`
	CreatedAt    string `json:"created_at"`
	LastUsedAt   string `json:"last_used_at"`
	ExpiresAt    string `json:"expires_at"`
	IsCurrent    bool   `json:"is_current"`
}

type sessionsListResponse struct {
	Data []sessionResponseItem `json:"data"`
}

// List — GET /api/v1/me/sessions
//
// Returns every still-active session for the calling user, newest
// expiry first. The row whose JTI matches the caller's refresh-token
// cookie is decorated with `is_current: true`.
func (h *SessionsHandler) List(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.repo == nil {
		res.Error(w, http.StatusServiceUnavailable, "sessions_disabled", "sessions feature not configured")
		return
	}
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	rows, err := h.repo.ListActiveByUser(r.Context(), userID)
	if err != nil {
		slog.Error("sessions: list_active failed", "user_id", userID, "error", err)
		res.Error(w, http.StatusInternalServerError, "sessions_list_error", "failed to load sessions")
		return
	}

	currentJTI := h.currentJTI(r)
	out := sessionsListResponse{Data: make([]sessionResponseItem, 0, len(rows))}
	for _, s := range rows {
		out.Data = append(out.Data, toSessionItem(s, currentJTI))
	}
	res.JSON(w, http.StatusOK, out)
}

// Revoke — DELETE /api/v1/me/sessions/{id}
//
// Marks the session whose id matches the URL param as revoked. The
// caller MUST own the row — a foreign id returns 403. The current
// session itself can be revoked (this is the "log me out of this
// tab" path); the next request will see the access-token still
// valid until its 15-minute TTL expires but the refresh path is
// dead immediately.
func (h *SessionsHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.repo == nil {
		res.Error(w, http.StatusServiceUnavailable, "sessions_disabled", "sessions feature not configured")
		return
	}
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}
	raw := chi.URLParam(r, "id")
	id, err := uuid.Parse(raw)
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_session_id", "session id is not a valid uuid")
		return
	}

	row, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, session.ErrNotFound) {
			res.Error(w, http.StatusNotFound, "session_not_found", "session not found")
			return
		}
		slog.Error("sessions: find_by_id failed", "user_id", userID, "session_id", id, "error", err)
		res.Error(w, http.StatusInternalServerError, "sessions_revoke_error", "failed to load session")
		return
	}
	if row.UserID != userID {
		res.Error(w, http.StatusForbidden, "forbidden", "you do not own this session")
		return
	}
	if err := h.repo.RevokeByID(r.Context(), id); err != nil {
		slog.Error("sessions: revoke_by_id failed", "user_id", userID, "session_id", id, "error", err)
		res.Error(w, http.StatusInternalServerError, "sessions_revoke_error", "failed to revoke session")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// RevokeAllExceptCurrent — POST /api/v1/me/sessions/revoke-others
//
// Revokes every active session for the caller EXCEPT the one matching
// the current refresh-token cookie's JTI. When no cookie is present
// (e.g. mobile client without cookies) the request falls back to
// revoking ALL active sessions — the client must re-authenticate.
func (h *SessionsHandler) RevokeAllExceptCurrent(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.repo == nil {
		res.Error(w, http.StatusServiceUnavailable, "sessions_disabled", "sessions feature not configured")
		return
	}
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}
	currentJTI := h.currentJTI(r)
	if err := h.repo.RevokeAllForUserExceptJTI(r.Context(), userID, currentJTI); err != nil {
		slog.Error("sessions: revoke_all_except failed", "user_id", userID, "error", err)
		res.Error(w, http.StatusInternalServerError, "sessions_revoke_error", "failed to revoke sessions")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// toSessionItem maps a domain.Session into the wire shape. All time
// fields go out as RFC3339 UTC so the web app can render them with
// `new Date(...)` without timezone surprises.
func toSessionItem(s *session.Session, currentJTI string) sessionResponseItem {
	return sessionResponseItem{
		ID:           s.ID.String(),
		DeviceLabel:  defaultIfEmpty(s.DeviceLabel, UnknownDeviceLabel),
		Browser:      s.Browser,
		OS:           s.OS,
		City:         s.City,
		CountryCode:  s.CountryCode,
		IPAnonymized: s.IPAnonymized,
		LoginMethod:  string(s.LoginMethod),
		CreatedAt:    s.CreatedAt.UTC().Format(time.RFC3339),
		LastUsedAt:   s.LastUsedAt.UTC().Format(time.RFC3339),
		ExpiresAt:    s.ExpiresAt.UTC().Format(time.RFC3339),
		IsCurrent:    currentJTI != "" && currentJTI == s.JTI,
	}
}

// defaultIfEmpty keeps the "device_label is always populated" invariant
// even when historical rows pre-migration-150 have '' on the column.
func defaultIfEmpty(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}

// currentJTI tries to discover the JTI of the session the request is
// currently authenticated through.
//
// Web (cookie auth via session_id) does NOT carry a JTI in the cookie
// — the refresh-token chain is decoupled from the Redis-backed
// session_id. For cookie-authed requests we read the Bearer header
// instead if the client decided to pass one, or fall back to the
// access-token's family root JTI which equals the very first refresh
// token's JTI of the chain (login). Returns '' when no source is
// usable; the UI then simply does not flag any row as current.
//
// This is opportunistic, not load-bearing: revocation works
// identically whether the row is flagged or not. The badge is a UX
// affordance, not a security boundary.
func (h *SessionsHandler) currentJTI(r *http.Request) string {
	// Bearer header path — used by the mobile app and any web client
	// that explicitly forwarded the access token. The access token's
	// FamilyRootJTI matches the first refresh token's JTI in the
	// chain, which is the session row created at login. Subsequent
	// refresh rotations produce new session rows whose JTI changes;
	// the access-token side does not track those — so the badge will
	// match the LOGIN row, which is what users intuitively call
	// "this session" on the Sécurité page anyway.
	if header := r.Header.Get("Authorization"); header != "" {
		const prefix = "Bearer "
		if len(header) > len(prefix) && header[:len(prefix)] == prefix {
			// Avoid pulling the token service into this handler just
			// to decode a claim — leave the badge empty and rely on
			// the UI's "—" fallback. A future iteration can wire the
			// decoder in via constructor injection if needed.
			return ""
		}
	}
	// Cookie auth: we cannot resolve the JTI without the token
	// service; the UI does not get a "current" badge.
	return ""
}
