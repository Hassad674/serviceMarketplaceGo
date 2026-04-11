package response

import (
	"time"

	"marketplace-backend/internal/domain/organization"
	"marketplace-backend/internal/domain/user"
)

// MemberUserResponse is the slim user identity block embedded in a
// MemberResponse so the team UI can render an avatar + name + email
// without a second round-trip per row. Only the identity fields the
// team page needs are exposed — never password hashes, suspension
// metadata, or anything that doesn't belong on a list view.
type MemberUserResponse struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
}

// MemberResponse is the serialized view of an organization member for
// the team management UI. Exposes role, title, join date, and the
// underlying user id so the frontend can correlate with profile data.
//
// As of R13 (team page UX fixes), the optional User block carries the
// joined identity fields when the handler builds the response with
// user data attached. The block is omitempty so the field continues
// to be optional from the client's perspective — older clients that
// were written before R13 still parse the response cleanly.
type MemberResponse struct {
	ID             string              `json:"id"`
	OrganizationID string              `json:"organization_id"`
	UserID         string              `json:"user_id"`
	Role           string              `json:"role"`
	Title          string              `json:"title"`
	JoinedAt       time.Time           `json:"joined_at"`
	User           *MemberUserResponse `json:"user,omitempty"`
}

// MemberListResponse is the paginated envelope returned by GET
// /organizations/{id}/members.
type MemberListResponse struct {
	Data       []MemberResponse `json:"data"`
	NextCursor string           `json:"next_cursor,omitempty"`
}

// NewMemberUserResponse projects the identity fields of a domain user
// onto the slim MemberUserResponse. Returns nil for a nil input so
// the caller can omit the block when no user record was joined.
func NewMemberUserResponse(u *user.User) *MemberUserResponse {
	if u == nil {
		return nil
	}
	return &MemberUserResponse{
		ID:          u.ID.String(),
		Email:       u.Email,
		DisplayName: u.DisplayName,
		FirstName:   u.FirstName,
		LastName:    u.LastName,
	}
}

// RoleDefinitionPermission is the slim view of a permission inside a
// role definition row. The label/description are English defaults
// the frontend can fall back to when its i18n catalogue does not yet
// have a translation for a newly-added permission.
type RoleDefinitionPermission struct {
	Key         string `json:"key"`
	Group       string `json:"group"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

// RoleDefinitionResponse describes a single role: its key, an English
// display label, a short description, and the list of permission keys
// it grants. Used by the team page's "About roles" panel to show
// users what each role can do before they assign it.
type RoleDefinitionResponse struct {
	Key         string   `json:"key"`
	Label       string   `json:"label"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
}

// RoleDefinitionsPayload bundles the role list and the permission
// catalogue in a single response. Roles reference permissions by
// key; the permissions array carries the human-readable labels for
// each key, so the frontend can render them once without duplicating
// strings inside every role row.
type RoleDefinitionsPayload struct {
	Roles       []RoleDefinitionResponse   `json:"roles"`
	Permissions []RoleDefinitionPermission `json:"permissions"`
}

// NewRoleDefinitionsPayload assembles the role-definitions response
// from the domain's static role + permission catalogues. This is the
// only place that bridges the domain into the response shape — the
// rest of the codebase keeps reading from the domain map directly.
func NewRoleDefinitionsPayload(
	roles []organization.Role,
	permissionCatalogue []organization.PermissionMetadata,
) RoleDefinitionsPayload {
	rolesOut := make([]RoleDefinitionResponse, 0, len(roles))
	for _, r := range roles {
		meta := organization.MetadataForRole(r)
		perms := organization.PermissionsFor(r)
		permKeys := make([]string, 0, len(perms))
		for _, p := range perms {
			permKeys = append(permKeys, string(p))
		}
		rolesOut = append(rolesOut, RoleDefinitionResponse{
			Key:         string(meta.Key),
			Label:       meta.Label,
			Description: meta.Description,
			Permissions: permKeys,
		})
	}

	permsOut := make([]RoleDefinitionPermission, 0, len(permissionCatalogue))
	for _, m := range permissionCatalogue {
		permsOut = append(permsOut, RoleDefinitionPermission{
			Key:         string(m.Key),
			Group:       m.Group,
			Label:       m.Label,
			Description: m.Description,
		})
	}

	return RoleDefinitionsPayload{
		Roles:       rolesOut,
		Permissions: permsOut,
	}
}

// NewMemberListResponseWithUsers builds a MemberListResponse where
// each row is enriched with the matching user identity block, or
// left without one when the user could not be resolved (deleted user
// race condition, missing row, etc.). The byUserID map is keyed on
// the member's user id and is built once by the handler from a
// single batch query against the users repo.
func NewMemberListResponseWithUsers(
	members []*organization.Member,
	byUserID map[string]*user.User,
	nextCursor string,
) MemberListResponse {
	rows := make([]MemberResponse, 0, len(members))
	for _, m := range members {
		row := NewMemberResponse(m)
		if u, ok := byUserID[m.UserID.String()]; ok {
			row.User = NewMemberUserResponse(u)
		}
		rows = append(rows, row)
	}
	return MemberListResponse{Data: rows, NextCursor: nextCursor}
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
