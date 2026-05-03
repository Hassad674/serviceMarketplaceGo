package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"marketplace-backend/internal/domain/organization"
	"marketplace-backend/internal/handler/dto/response"
	jsondec "marketplace-backend/pkg/decode"
	res "marketplace-backend/pkg/response"
)

// Phase 6 admin team endpoints. Mounted under /api/v1/admin/* and
// gated by the existing RequireAdmin middleware — no additional
// authorization is done here.

// GetUserOrganization handles GET /api/v1/admin/users/{id}/organization.
//
// Returns the full team detail for a user: their org, the member list,
// pending invitations, and the pending transfer state (if any).
// Responds with 404 when the user has no org (solo Provider or
// unprovisioned account).
func (h *AdminHandler) GetUserOrganization(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_id", "id must be a valid UUID")
		return
	}

	detail, err := h.svc.GetUserOrganizationDetail(r.Context(), userID)
	if err != nil {
		if errors.Is(err, organization.ErrOrgNotFound) {
			res.Error(w, http.StatusNotFound, "org_not_found", "this user has no organization")
			return
		}
		slog.Error("admin get user organization", "error", err, "user_id", userID.String())
		res.Error(w, http.StatusInternalServerError, "internal_error", "failed to load organization")
		return
	}

	res.JSON(w, http.StatusOK, response.NewAdminOrganizationDetailResponse(detail))
}

// ForceTransferOwnership handles POST /api/v1/admin/organizations/{id}/force-transfer.
//
// Body: { target_user_id: UUID }. The target MUST already be a member
// of the org but can hold any role (including non-Admin). The override
// bypasses the usual "target must be Admin" guardrail so platform
// admins can recover a locked organization.
func (h *AdminHandler) ForceTransferOwnership(w http.ResponseWriter, r *http.Request) {
	orgID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_id", "id must be a valid UUID")
		return
	}

	var body struct {
		TargetUserID string `json:"target_user_id"`
	}
	// F.5 B1: bound + reject unknown fields. Tiny ID-only payload.
	if err := jsondec.DecodeBody(w, r, &body, 4<<10); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_request", "malformed JSON body")
		return
	}
	targetID, err := uuid.Parse(body.TargetUserID)
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_target", "target_user_id must be a valid UUID")
		return
	}

	org, err := h.svc.ForceTransferOwnership(r.Context(), orgID, targetID)
	if err != nil {
		handleAdminTeamError(w, err, "force transfer ownership")
		return
	}

	res.JSON(w, http.StatusOK, map[string]string{
		"organization_id": org.ID.String(),
		"new_owner_id":    org.OwnerUserID.String(),
	})
}

// ForceUpdateMemberRole handles PATCH /api/v1/admin/organizations/{id}/members/{userID}.
//
// Body: { role: "admin" | "member" | "viewer" }. Changing to "owner"
// is rejected — use /force-transfer instead.
func (h *AdminHandler) ForceUpdateMemberRole(w http.ResponseWriter, r *http.Request) {
	orgID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_id", "id must be a valid UUID")
		return
	}
	targetID, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_target", "userID must be a valid UUID")
		return
	}

	var body struct {
		Role string `json:"role"`
	}
	// F.5 B1: bound + reject unknown fields.
	if err := jsondec.DecodeBody(w, r, &body, 4<<10); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_request", "malformed JSON body")
		return
	}
	role := organization.Role(body.Role)

	member, err := h.svc.ForceUpdateMemberRole(r.Context(), orgID, targetID, role)
	if err != nil {
		handleAdminTeamError(w, err, "force update member role")
		return
	}

	res.JSON(w, http.StatusOK, response.AdminOrganizationMemberResponse{
		ID:             member.ID.String(),
		OrganizationID: member.OrganizationID.String(),
		UserID:         member.UserID.String(),
		Role:           string(member.Role),
		Title:          member.Title,
		JoinedAt:       member.JoinedAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

// ForceRemoveMember handles DELETE /api/v1/admin/organizations/{id}/members/{userID}.
//
// Cannot remove the current Owner — the admin must use /force-transfer
// first to move ownership off the target.
func (h *AdminHandler) ForceRemoveMember(w http.ResponseWriter, r *http.Request) {
	orgID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_id", "id must be a valid UUID")
		return
	}
	targetID, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_target", "userID must be a valid UUID")
		return
	}

	if err := h.svc.ForceRemoveMember(r.Context(), orgID, targetID); err != nil {
		handleAdminTeamError(w, err, "force remove member")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ForceCancelInvitation handles DELETE /api/v1/admin/organizations/{id}/invitations/{invID}.
//
// Idempotent: already-accepted or already-cancelled invitations are
// treated as no-ops and return 204 regardless.
func (h *AdminHandler) ForceCancelInvitation(w http.ResponseWriter, r *http.Request) {
	// orgID is only used for URL scoping — the cancel itself is keyed
	// on invitation_id which is globally unique. We still require it
	// in the URL so the frontend can't craft arbitrary inv IDs.
	if _, err := uuid.Parse(chi.URLParam(r, "id")); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_id", "id must be a valid UUID")
		return
	}
	invID, err := uuid.Parse(chi.URLParam(r, "invID"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_invitation", "invID must be a valid UUID")
		return
	}

	if err := h.svc.ForceCancelInvitation(r.Context(), invID); err != nil {
		handleAdminTeamError(w, err, "force cancel invitation")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleAdminTeamError maps the narrow set of domain errors these
// endpoints can return to their HTTP counterparts. Unknown errors
// are logged and surfaced as 500 with a generic message.
func handleAdminTeamError(w http.ResponseWriter, err error, op string) {
	switch {
	case errors.Is(err, organization.ErrOrgNotFound):
		res.Error(w, http.StatusNotFound, "org_not_found", "organization not found")
	case errors.Is(err, organization.ErrMemberNotFound):
		res.Error(w, http.StatusNotFound, "member_not_found", "member not found")
	case errors.Is(err, organization.ErrInvitationNotFound):
		res.Error(w, http.StatusNotFound, "invitation_not_found", "invitation not found")
	case errors.Is(err, organization.ErrOwnerCannotBeRemoved):
		res.Error(w, http.StatusConflict, "owner_cannot_be_removed", "cannot remove the current owner")
	case errors.Is(err, organization.ErrInvalidRole),
		errors.Is(err, organization.ErrCannotInviteAsOwner):
		res.Error(w, http.StatusBadRequest, "invalid_role", "invalid target role")
	case errors.Is(err, organization.ErrPermissionDenied):
		res.Error(w, http.StatusConflict, "conflict", "operation not allowed in the current state")
	case errors.Is(err, organization.ErrTransferTargetInvalid):
		res.Error(w, http.StatusBadRequest, "invalid_transfer_target", "target user is not a member of this org")
	default:
		slog.Error("admin team endpoint failed", "op", op, "error", err)
		res.Error(w, http.StatusInternalServerError, "internal_error", "failed to complete operation")
	}
}
