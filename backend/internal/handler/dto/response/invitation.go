package response

import (
	"time"

	"marketplace-backend/internal/domain/organization"
)

// InvitationResponse is the serialized view of a persisted team
// invitation sent by the API. Omits the secret token — the token is
// only visible via the email link and must never leak into an API
// response body.
type InvitationResponse struct {
	ID              string    `json:"id"`
	OrganizationID  string    `json:"organization_id"`
	Email           string    `json:"email"`
	FirstName       string    `json:"first_name"`
	LastName        string    `json:"last_name"`
	Title           string    `json:"title"`
	Role            string    `json:"role"`
	Status          string    `json:"status"`
	InvitedByUserID string    `json:"invited_by_user_id"`
	ExpiresAt       time.Time `json:"expires_at"`
	CreatedAt       time.Time `json:"created_at"`
}

// InvitationListResponse is the paginated envelope for GET
// /organizations/{id}/invitations.
type InvitationListResponse struct {
	Data       []InvitationResponse `json:"data"`
	NextCursor string               `json:"next_cursor,omitempty"`
}

// InvitationPreviewResponse is the public view returned by the
// /invitations/validate?token=X endpoint. Contains the minimum the
// acceptance page needs to render the form without exposing internal
// ids beyond the org and the invitation itself.
type InvitationPreviewResponse struct {
	ID               string    `json:"id"`
	OrganizationID   string    `json:"organization_id"`
	OrganizationName string    `json:"organization_name"`
	OrganizationType string    `json:"organization_type"`
	Email            string    `json:"email"`
	FirstName        string    `json:"first_name"`
	LastName         string    `json:"last_name"`
	Title            string    `json:"title"`
	Role             string    `json:"role"`
	ExpiresAt        time.Time `json:"expires_at"`
}

func NewInvitationResponse(inv *organization.Invitation) InvitationResponse {
	return InvitationResponse{
		ID:              inv.ID.String(),
		OrganizationID:  inv.OrganizationID.String(),
		Email:           inv.Email,
		FirstName:       inv.FirstName,
		LastName:        inv.LastName,
		Title:           inv.Title,
		Role:            inv.Role.String(),
		Status:          string(inv.Status),
		InvitedByUserID: inv.InvitedByUserID.String(),
		ExpiresAt:       inv.ExpiresAt,
		CreatedAt:       inv.CreatedAt,
	}
}

func NewInvitationListResponse(items []*organization.Invitation, nextCursor string) InvitationListResponse {
	data := make([]InvitationResponse, 0, len(items))
	for _, inv := range items {
		data = append(data, NewInvitationResponse(inv))
	}
	return InvitationListResponse{Data: data, NextCursor: nextCursor}
}

// NewInvitationPreviewResponse builds the public preview. The
// organization name is the Owner's display_name (same convention used
// elsewhere) — the handler is expected to have already loaded the org.
// For the name, the caller currently passes the orgType; a dedicated
// orgName field can be added later if we want to split the two.
func NewInvitationPreviewResponse(inv *organization.Invitation, org *organization.Organization) InvitationPreviewResponse {
	return InvitationPreviewResponse{
		ID:               inv.ID.String(),
		OrganizationID:   org.ID.String(),
		OrganizationName: "", // populated by the handler if a dedicated name source is wired later
		OrganizationType: org.Type.String(),
		Email:            inv.Email,
		FirstName:        inv.FirstName,
		LastName:         inv.LastName,
		Title:            inv.Title,
		Role:             inv.Role.String(),
		ExpiresAt:        inv.ExpiresAt,
	}
}
