package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	orgapp "marketplace-backend/internal/app/organization"
	"marketplace-backend/internal/domain/organization"
	"marketplace-backend/internal/handler/dto/response"
	"marketplace-backend/internal/handler/middleware"
	"marketplace-backend/pkg/validator"

	res "marketplace-backend/pkg/response"
)

// RoleOverridesHandler owns the HTTP surface for the per-org role
// permissions editor: read the customized matrix and save changes.
//
// Authorization rules:
//
//   - GET /organizations/{orgID}/role-permissions
//     Any authenticated member of the org (read-only view for every role).
//
//   - PATCH /organizations/{orgID}/role-permissions
//     Owner only. Enforced AT BOTH the middleware fast-path
//     (RequirePermission(PermTeamManageRolePermissions), which is
//     Owner-only by default and non-overridable) AND at the service
//     layer (see RoleOverridesService.UpdateRoleOverrides) — defense
//     in depth.
type RoleOverridesHandler struct {
	service *orgapp.RoleOverridesService
}

func NewRoleOverridesHandler(svc *orgapp.RoleOverridesService) *RoleOverridesHandler {
	return &RoleOverridesHandler{service: svc}
}

// ---------------------------------------------------------------------------
// GET /organizations/{orgID}/role-permissions
// ---------------------------------------------------------------------------

// GetMatrix returns the full customized permission matrix for the
// org. Every role row (Owner, Admin, Member, Viewer) is included,
// with the Owner row marked as locked+read-only so the UI can render
// it accordingly.
func (h *RoleOverridesHandler) GetMatrix(w http.ResponseWriter, r *http.Request) {
	actorID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	orgID, err := uuid.Parse(chi.URLParam(r, "orgID"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_org_id", "invalid organization id")
		return
	}

	matrix, err := h.service.GetMatrix(r.Context(), actorID, orgID)
	if err != nil {
		h.handleRoleOverridesError(w, err)
		return
	}
	res.JSON(w, http.StatusOK, response.NewRolePermissionsMatrixResponse(matrix))
}

// ---------------------------------------------------------------------------
// PATCH /organizations/{orgID}/role-permissions
// ---------------------------------------------------------------------------

// updateRoleOverridesRequest is the wire shape of the save payload.
//
// The overrides map is the FULL desired state for the target role —
// any previous override not present here reverts to the default.
// This matches the "replace" semantics of the service layer.
type updateRoleOverridesRequest struct {
	Role      string          `json:"role"`
	Overrides map[string]bool `json:"overrides"`
}

// UpdateMatrix handles the Owner-only write for the role-permissions
// editor. The body is JSON:
//
//	{
//	  "role": "admin",
//	  "overrides": {
//	    "billing.manage": true,
//	    "jobs.delete": false
//	  }
//	}
//
// Returns the same response shape as GET so the frontend can refresh
// its cache with a single PATCH round-trip.
func (h *RoleOverridesHandler) UpdateMatrix(w http.ResponseWriter, r *http.Request) {
	actorID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	orgID, err := uuid.Parse(chi.URLParam(r, "orgID"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_org_id", "invalid organization id")
		return
	}

	var req updateRoleOverridesRequest
	if err := validator.DecodeJSON(r, &req); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	role := organization.Role(strings.TrimSpace(strings.ToLower(req.Role)))
	if !role.IsValid() {
		res.Error(w, http.StatusBadRequest, "invalid_role", "invalid role")
		return
	}

	// Convert the string-keyed wire map into the domain-typed map
	// that the service expects. Unknown keys are passed through
	// unchanged — the service's ValidateRoleOverrides rejects them
	// with a clean error code.
	overrides := make(map[organization.Permission]bool, len(req.Overrides))
	for k, v := range req.Overrides {
		overrides[organization.Permission(strings.TrimSpace(k))] = v
	}

	result, err := h.service.UpdateRoleOverrides(r.Context(), orgapp.UpdateRoleOverridesInput{
		ActorUserID:    actorID,
		OrganizationID: orgID,
		Role:           role,
		Overrides:      overrides,
		IPAddress:      clientIP(r),
	})
	if err != nil {
		h.handleRoleOverridesError(w, err)
		return
	}

	// Refresh the matrix so the frontend can update its cached view
	// without a second round-trip. The GetMatrix call is cheap and
	// keeps the client and server in lock-step.
	matrix, getErr := h.service.GetMatrix(r.Context(), actorID, orgID)
	if getErr != nil {
		slog.Warn("role permissions update: matrix refresh failed", "error", getErr)
		// Still return a success — the save landed.
		res.JSON(w, http.StatusOK, response.NewRolePermissionsUpdateResponse(result, nil))
		return
	}
	res.JSON(w, http.StatusOK, response.NewRolePermissionsUpdateResponse(result, matrix))
}

// ---------------------------------------------------------------------------
// Error mapping
// ---------------------------------------------------------------------------

// handleRoleOverridesError maps domain sentinels to HTTP responses.
// Uses errors.Is so wrapped errors surface correctly.
func (h *RoleOverridesHandler) handleRoleOverridesError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, organization.ErrNotAMember):
		res.Error(w, http.StatusForbidden, "not_a_member", "you are not a member of this organization")
	case errors.Is(err, organization.ErrPermissionDenied):
		res.Error(w, http.StatusForbidden, "permission_denied", "only the Owner can edit role permissions")
	case errors.Is(err, organization.ErrCannotOverrideOwner):
		res.Error(w, http.StatusBadRequest, "cannot_override_owner", "the Owner role cannot be customized")
	case errors.Is(err, organization.ErrPermissionNotOverridable):
		res.Error(w, http.StatusBadRequest, "permission_not_overridable", "this permission cannot be customized — it is locked for security reasons")
	case errors.Is(err, organization.ErrUnknownPermission):
		res.Error(w, http.StatusBadRequest, "unknown_permission", "one or more permission keys are unknown")
	case errors.Is(err, organization.ErrInvalidRole):
		res.Error(w, http.StatusBadRequest, "invalid_role", "invalid role")
	case errors.Is(err, organization.ErrRolePermChangesRateLimit):
		w.Header().Set("Retry-After", "3600")
		res.Error(w, http.StatusTooManyRequests, "rate_limited", "too many role permission changes today — try again tomorrow")
	case errors.Is(err, organization.ErrOrgNotFound):
		res.Error(w, http.StatusNotFound, "organization_not_found", "organization not found")
	default:
		slog.Error("unhandled role overrides error", "error", err)
		res.Error(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
	}
}

// clientIP extracts the caller's IP from the request. Prefers the
// X-Forwarded-For header (trusted in prod because the app sits
// behind a load balancer) and falls back to RemoteAddr.
func clientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		// X-Forwarded-For can carry a list — first entry is the client.
		if idx := strings.Index(fwd, ","); idx > 0 {
			return strings.TrimSpace(fwd[:idx])
		}
		return strings.TrimSpace(fwd)
	}
	// r.RemoteAddr is host:port — strip the port.
	addr := r.RemoteAddr
	if idx := strings.LastIndex(addr, ":"); idx > 0 {
		return addr[:idx]
	}
	return addr
}
