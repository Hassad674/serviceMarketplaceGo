package handler

import (
	"errors"
	"log/slog"
	"net"
	"net/http"
	"strings"

	"github.com/google/uuid"

	consentapp "marketplace-backend/internal/app/consent"
	"marketplace-backend/internal/domain/consent"
	"marketplace-backend/internal/handler/middleware"
	jsondec "marketplace-backend/pkg/decode"
	res "marketplace-backend/pkg/response"
)

// ConsentHandler exposes the single endpoint POST /consent/log used to
// record server-side proof of consent decisions made on the cookie
// banner. Authentication is OPTIONAL: anonymous visitors must be able
// to record their refusal/acceptance just as authenticated members do.
//
// The handler is intentionally small — derive IP + UA from the
// request, hand the input to the service, return 204 on success.
type ConsentHandler struct {
	svc *consentapp.Service
}

// NewConsentHandler wires the handler to the consent app service.
// The service is the only collaborator — RBAC / ownership checks do
// NOT apply because consent is a property of the visitor, not of any
// owned resource.
func NewConsentHandler(svc *consentapp.Service) *ConsentHandler {
	return &ConsentHandler{svc: svc}
}

// recordConsentRequest is the JSON body shape accepted by Log. The
// handler never trusts a client-supplied IP / UA — those are derived
// from the HTTP transport layer. Categories must be a non-empty array
// and Action must match the domain enum (validated downstream).
type recordConsentRequest struct {
	Categories []string `json:"categories"`
	Action     string   `json:"action"`
	SessionID  string   `json:"session_id,omitempty"`
}

// Log records a consent decision. Body shape is small (≤ 1 KiB) so
// the DecodeBody cap is tight to short-circuit DoS attempts.
//
//	POST /api/v1/consent/log
//	{ "action": "accept_all", "categories": ["analytics"], "session_id": "..." }
//
// Returns 204 on success, 400 on validation, 500 on persistence error.
func (h *ConsentHandler) Log(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil {
		res.Error(w, http.StatusServiceUnavailable, "feature_disabled",
			"consent logging is not configured")
		return
	}
	var req recordConsentRequest
	if err := jsondec.DecodeBody(w, r, &req, 1<<10); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}

	var userIDPtr *uuid.UUID
	if uid, ok := middleware.GetUserID(r.Context()); ok && uid != uuid.Nil {
		copied := uid
		userIDPtr = &copied
	}

	in := consentapp.RecordInput{
		UserID:     userIDPtr,
		SessionID:  strings.TrimSpace(req.SessionID),
		Categories: req.Categories,
		Action:     consent.Action(strings.TrimSpace(req.Action)),
		RawIP:      remoteIPFromRequest(r),
		UserAgent:  r.UserAgent(),
	}

	if _, err := h.svc.Record(r.Context(), in); err != nil {
		switch {
		case errors.Is(err, consent.ErrInvalidAction),
			errors.Is(err, consent.ErrCategoriesRequired),
			errors.Is(err, consent.ErrIPAnonymizedRequired),
			errors.Is(err, consent.ErrUserAgentHashRequired):
			res.Error(w, http.StatusBadRequest, "invalid_input", err.Error())
		default:
			slog.Error("consent log persist", "error", err.Error())
			res.Error(w, http.StatusInternalServerError, "internal_error",
				"could not record consent")
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// remoteIPFromRequest extracts the leftmost public IP from
// X-Forwarded-For when present, otherwise falls back to RemoteAddr.
// Public so the handler can hand the raw IP to the domain truncator
// (gdpr.TruncateIP) — no normalisation here.
//
// Returns the empty string when nothing parseable is available; the
// service will then fail validation with ErrIPAnonymizedRequired.
func remoteIPFromRequest(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		for _, candidate := range strings.Split(xff, ",") {
			candidate = strings.TrimSpace(candidate)
			if ip := net.ParseIP(candidate); ip != nil {
				return ip.String()
			}
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.String()
	}
	return strings.TrimSpace(host)
}
