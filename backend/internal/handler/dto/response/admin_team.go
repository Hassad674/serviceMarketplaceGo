package response

import (
	"time"

	"marketplace-backend/internal/app/admin"
	"marketplace-backend/internal/domain/organization"
)

// AdminOrganizationDetailResponse is the JSON payload returned by
// GET /api/v1/admin/users/{id}/organization. It bundles everything
// the admin UI needs to render the team section of a user detail
// page in a single response (no N+1).

type AdminOrganizationResponse struct {
	ID                         string  `json:"id"`
	Type                       string  `json:"type"`
	OwnerUserID                string  `json:"owner_user_id"`
	PendingTransferToUserID    *string `json:"pending_transfer_to_user_id,omitempty"`
	PendingTransferInitiatedAt *string `json:"pending_transfer_initiated_at,omitempty"`
	PendingTransferExpiresAt   *string `json:"pending_transfer_expires_at,omitempty"`
	CreatedAt                  string  `json:"created_at"`
	UpdatedAt                  string  `json:"updated_at"`
}

type AdminOrganizationMemberResponse struct {
	ID             string `json:"id"`
	OrganizationID string `json:"organization_id"`
	UserID         string `json:"user_id"`
	Role           string `json:"role"`
	Title          string `json:"title"`
	JoinedAt       string `json:"joined_at"`
}

type AdminOrganizationInvitationResponse struct {
	ID              string  `json:"id"`
	OrganizationID  string  `json:"organization_id"`
	Email           string  `json:"email"`
	FirstName       string  `json:"first_name"`
	LastName        string  `json:"last_name"`
	Title           string  `json:"title"`
	Role            string  `json:"role"`
	InvitedByUserID string  `json:"invited_by_user_id"`
	Status          string  `json:"status"`
	ExpiresAt       string  `json:"expires_at"`
	AcceptedAt      *string `json:"accepted_at,omitempty"`
	CancelledAt     *string `json:"cancelled_at,omitempty"`
	CreatedAt       string  `json:"created_at"`
}

type AdminOrganizationDetailResponse struct {
	Organization       AdminOrganizationResponse             `json:"organization"`
	Members            []AdminOrganizationMemberResponse     `json:"members"`
	PendingInvitations []AdminOrganizationInvitationResponse `json:"pending_invitations"`
	// ViewingRole tells the admin UI which role the user that started
	// the lookup holds in the org. "owner" for Owners; "admin" /
	// "member" / "viewer" for operators. The UI uses this to render
	// a breadcrumb or highlight the user's row in the members list.
	ViewingRole string `json:"viewing_role"`
}

// NewAdminOrganizationDetailResponse converts an app-layer detail
// aggregate into the HTTP response shape.
func NewAdminOrganizationDetailResponse(d *admin.AdminOrganizationDetail) AdminOrganizationDetailResponse {
	return AdminOrganizationDetailResponse{
		Organization:       newAdminOrganizationResponse(d.Organization),
		Members:            newAdminMemberList(d.Members),
		PendingInvitations: newAdminInvitationList(d.PendingInvitations),
		ViewingRole:        string(d.ViewingRole),
	}
}

func newAdminOrganizationResponse(org *organization.Organization) AdminOrganizationResponse {
	r := AdminOrganizationResponse{
		ID:          org.ID.String(),
		Type:        string(org.Type),
		OwnerUserID: org.OwnerUserID.String(),
		CreatedAt:   org.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   org.UpdatedAt.Format(time.RFC3339),
	}
	if org.PendingTransferToUserID != nil {
		s := org.PendingTransferToUserID.String()
		r.PendingTransferToUserID = &s
	}
	if org.PendingTransferInitiatedAt != nil {
		s := org.PendingTransferInitiatedAt.Format(time.RFC3339)
		r.PendingTransferInitiatedAt = &s
	}
	if org.PendingTransferExpiresAt != nil {
		s := org.PendingTransferExpiresAt.Format(time.RFC3339)
		r.PendingTransferExpiresAt = &s
	}
	return r
}

func newAdminMemberList(members []*organization.Member) []AdminOrganizationMemberResponse {
	out := make([]AdminOrganizationMemberResponse, 0, len(members))
	for _, m := range members {
		out = append(out, AdminOrganizationMemberResponse{
			ID:             m.ID.String(),
			OrganizationID: m.OrganizationID.String(),
			UserID:         m.UserID.String(),
			Role:           string(m.Role),
			Title:          m.Title,
			JoinedAt:       m.JoinedAt.Format(time.RFC3339),
		})
	}
	return out
}

func newAdminInvitationList(invitations []*organization.Invitation) []AdminOrganizationInvitationResponse {
	out := make([]AdminOrganizationInvitationResponse, 0, len(invitations))
	for _, inv := range invitations {
		r := AdminOrganizationInvitationResponse{
			ID:              inv.ID.String(),
			OrganizationID:  inv.OrganizationID.String(),
			Email:           inv.Email,
			FirstName:       inv.FirstName,
			LastName:        inv.LastName,
			Title:           inv.Title,
			Role:            string(inv.Role),
			InvitedByUserID: inv.InvitedByUserID.String(),
			Status:          string(inv.Status),
			ExpiresAt:       inv.ExpiresAt.Format(time.RFC3339),
			CreatedAt:       inv.CreatedAt.Format(time.RFC3339),
		}
		if inv.AcceptedAt != nil {
			s := inv.AcceptedAt.Format(time.RFC3339)
			r.AcceptedAt = &s
		}
		if inv.CancelledAt != nil {
			s := inv.CancelledAt.Format(time.RFC3339)
			r.CancelledAt = &s
		}
		out = append(out, r)
	}
	return out
}
