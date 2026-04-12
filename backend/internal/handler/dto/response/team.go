package response

import (
	"time"

	orgapp "marketplace-backend/internal/app/organization"
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
//
// Overridable is true when the permission can be toggled per-org via
// the role-permissions editor. False means the permission is locked
// Owner-only forever (PermOrgDelete, PermWalletWithdraw, …). The
// frontend renders locked permissions with a tooltip explaining why
// they cannot be customized.
type RoleDefinitionPermission struct {
	Key         string `json:"key"`
	Group       string `json:"group"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Overridable bool   `json:"overridable"`
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
			Overridable: organization.IsOverridable(m.Key),
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

// ---------------------------------------------------------------------------
// Role permissions editor (R17 — per-org customization)
// ---------------------------------------------------------------------------

// RolePermissionCell is one (permission, state) cell in the matrix.
// The frontend uses the `state` field to pick the right visual
// treatment: default (no badge), granted_override (green badge),
// revoked_override (red badge), locked (disabled + lock icon).
type RolePermissionCell struct {
	Key         string `json:"key"`
	Group       string `json:"group"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Granted     bool   `json:"granted"`
	State       string `json:"state"`
	Locked      bool   `json:"locked"`
}

// RolePermissionsRow is the catalogue of resolved permissions for one
// role, ready to render as a full column in the editor UI.
type RolePermissionsRow struct {
	Role        string               `json:"role"`
	Label       string               `json:"label"`
	Description string               `json:"description"`
	Permissions []RolePermissionCell `json:"permissions"`
}

// RolePermissionsMatrixResponse is the response body of
// GET /organizations/{id}/role-permissions. The top-level `roles`
// array is always in the canonical order (Owner, Admin, Member, Viewer)
// so the frontend can index into it without resorting.
type RolePermissionsMatrixResponse struct {
	Roles []RolePermissionsRow `json:"roles"`
}

// NewRolePermissionsMatrixResponse projects the service's matrix
// struct onto the wire shape. Permission metadata (label, description,
// group) is resolved from the domain catalogue so the backend stays
// the single source of truth for the human-readable labels.
func NewRolePermissionsMatrixResponse(matrix *orgapp.RolePermissionsMatrix) RolePermissionsMatrixResponse {
	if matrix == nil {
		return RolePermissionsMatrixResponse{Roles: []RolePermissionsRow{}}
	}
	rows := make([]RolePermissionsRow, 0, len(matrix.Roles))
	for _, r := range matrix.Roles {
		cells := make([]RolePermissionCell, 0, len(r.Permissions))
		for _, view := range r.Permissions {
			meta := organization.MetadataForPermission(view.Key)
			cells = append(cells, RolePermissionCell{
				Key:         string(view.Key),
				Group:       meta.Group,
				Label:       meta.Label,
				Description: meta.Description,
				Granted:     view.Granted,
				State:       string(view.State),
				Locked:      view.Locked,
			})
		}
		rows = append(rows, RolePermissionsRow{
			Role:        string(r.Role),
			Label:       r.Label,
			Description: r.Description,
			Permissions: cells,
		})
	}
	return RolePermissionsMatrixResponse{Roles: rows}
}

// RolePermissionsUpdateResponse is the response body of
// PATCH /organizations/{id}/role-permissions. Bundles the change
// summary (counts + the granted / revoked permission keys) and the
// refreshed matrix so the frontend cache stays in sync with one
// round-trip.
type RolePermissionsUpdateResponse struct {
	Role            string                         `json:"role"`
	GrantedKeys     []string                       `json:"granted_keys"`
	RevokedKeys     []string                       `json:"revoked_keys"`
	AffectedMembers int                            `json:"affected_members"`
	Matrix          *RolePermissionsMatrixResponse `json:"matrix,omitempty"`
}

// NewRolePermissionsUpdateResponse converts the service result + an
// optional refreshed matrix into the PATCH response shape.
func NewRolePermissionsUpdateResponse(
	result *orgapp.UpdateRoleOverridesResult,
	matrix *orgapp.RolePermissionsMatrix,
) RolePermissionsUpdateResponse {
	if result == nil {
		return RolePermissionsUpdateResponse{
			GrantedKeys: []string{},
			RevokedKeys: []string{},
		}
	}
	granted := make([]string, 0, len(result.GrantedKeys))
	for _, p := range result.GrantedKeys {
		granted = append(granted, string(p))
	}
	revoked := make([]string, 0, len(result.RevokedKeys))
	for _, p := range result.RevokedKeys {
		revoked = append(revoked, string(p))
	}
	resp := RolePermissionsUpdateResponse{
		Role:            string(result.Role),
		GrantedKeys:     granted,
		RevokedKeys:     revoked,
		AffectedMembers: result.AffectedMembers,
	}
	if matrix != nil {
		m := NewRolePermissionsMatrixResponse(matrix)
		resp.Matrix = &m
	}
	return resp
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
