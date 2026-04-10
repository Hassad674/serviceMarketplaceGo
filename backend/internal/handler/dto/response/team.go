package response

import (
	"time"

	"marketplace-backend/internal/domain/organization"
)

// MemberResponse is the serialized view of an organization member for
// the team management UI. Exposes role, title, join date, and the
// underlying user id so the frontend can correlate with profile data.
type MemberResponse struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	UserID         string    `json:"user_id"`
	Role           string    `json:"role"`
	Title          string    `json:"title"`
	JoinedAt       time.Time `json:"joined_at"`
}

// MemberListResponse is the paginated envelope returned by GET
// /organizations/{id}/members.
type MemberListResponse struct {
	Data       []MemberResponse `json:"data"`
	NextCursor string           `json:"next_cursor,omitempty"`
}

// TransferResponse is the envelope returned by transfer-ownership
// endpoints. Contains the pending transfer fields from the org so the
// frontend can render "transfer pending" banners or the target's
// acceptance prompt.
type TransferResponse struct {
	OrganizationID             string     `json:"organization_id"`
	CurrentOwnerUserID         string     `json:"current_owner_user_id"`
	PendingTransferToUserID    *string    `json:"pending_transfer_to_user_id,omitempty"`
	PendingTransferInitiatedAt *time.Time `json:"pending_transfer_initiated_at,omitempty"`
	PendingTransferExpiresAt   *time.Time `json:"pending_transfer_expires_at,omitempty"`
}

func NewMemberResponse(m *organization.Member) MemberResponse {
	if m == nil {
		return MemberResponse{}
	}
	return MemberResponse{
		ID:             m.ID.String(),
		OrganizationID: m.OrganizationID.String(),
		UserID:         m.UserID.String(),
		Role:           m.Role.String(),
		Title:          m.Title,
		JoinedAt:       m.JoinedAt,
	}
}

func NewMemberListResponse(members []*organization.Member, nextCursor string) MemberListResponse {
	data := make([]MemberResponse, 0, len(members))
	for _, m := range members {
		data = append(data, NewMemberResponse(m))
	}
	return MemberListResponse{Data: data, NextCursor: nextCursor}
}

func NewTransferResponse(org *organization.Organization) TransferResponse {
	if org == nil {
		return TransferResponse{}
	}
	resp := TransferResponse{
		OrganizationID:             org.ID.String(),
		CurrentOwnerUserID:         org.OwnerUserID.String(),
		PendingTransferInitiatedAt: org.PendingTransferInitiatedAt,
		PendingTransferExpiresAt:   org.PendingTransferExpiresAt,
	}
	if org.PendingTransferToUserID != nil {
		id := org.PendingTransferToUserID.String()
		resp.PendingTransferToUserID = &id
	}
	return resp
}
