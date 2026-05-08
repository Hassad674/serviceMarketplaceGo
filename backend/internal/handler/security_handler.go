package handler

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"marketplace-backend/internal/app/security"
	"marketplace-backend/internal/handler/middleware"
	res "marketplace-backend/pkg/response"
)

// SecurityHandler exposes the read-only "Activité récente" feed for the
// account/security tab. Auth is enforced upstream by the router; the
// rows the user sees are pinned to their own user_id by the underlying
// audit_logs repository (RLS + WHERE user_id = $1) — there is no
// cross-tenant query path here.
type SecurityHandler struct {
	svc *security.Service
}

// NewSecurityHandler returns a new handler. svc may be nil — when so,
// every handler method short-circuits with 503 so the rest of the API
// stays available even if the security feature is not wired.
func NewSecurityHandler(svc *security.Service) *SecurityHandler {
	return &SecurityHandler{svc: svc}
}

// securityActivityResponseItem is the shape one row takes on the wire.
// Naming mirrors the spec brief: type d'accès → access_kind +
// user_agent_summary, localisation → ip_address + country_hint.
type securityActivityResponseItem struct {
	ID               string `json:"id"`
	Action           string `json:"action"`
	IPAddress        string `json:"ip_address,omitempty"`
	UserAgentSummary string `json:"user_agent_summary"`
	AccessKind       string `json:"access_kind"`
	CountryHint      string `json:"country_hint,omitempty"`
	CreatedAt        string `json:"created_at"`
}

// securityActivityResponse is the cursor-paginated envelope.
type securityActivityResponse struct {
	Data       []securityActivityResponseItem `json:"data"`
	NextCursor string                         `json:"next_cursor,omitempty"`
}

// ListActivity — GET /api/v1/me/security/activity?cursor=&limit=
//
// Returns the most recent authentication-related audit events
// attributable to the calling user, newest-first. The list is
// strictly user-scoped: a member of an organization will NOT see
// another member's authentications even when they share an org.
func (h *SecurityHandler) ListActivity(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.svc == nil {
		res.Error(w, http.StatusServiceUnavailable, "security_disabled", "security activity feature not configured")
		return
	}
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	cursor := r.URL.Query().Get("cursor")
	limit := 20
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 {
			limit = v
		}
	}

	page, err := h.svc.ListActivity(r.Context(), userID, cursor, limit)
	if err != nil {
		slog.Error("security activity list failed", "user_id", userID, "error", err)
		res.Error(w, http.StatusInternalServerError, "security_activity_error", "failed to load security activity")
		return
	}

	out := securityActivityResponse{
		Data:       make([]securityActivityResponseItem, 0, len(page.Events)),
		NextCursor: page.NextCursor,
	}
	for _, ev := range page.Events {
		out.Data = append(out.Data, securityActivityResponseItem{
			ID:               ev.ID.String(),
			Action:           string(ev.Action),
			IPAddress:        ev.IPAddress,
			UserAgentSummary: ev.UserAgentSummary.Display,
			AccessKind:       string(ev.UserAgentSummary.Kind),
			CountryHint:      ev.CountryHint,
			CreatedAt:        ev.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	res.JSON(w, http.StatusOK, out)
}
