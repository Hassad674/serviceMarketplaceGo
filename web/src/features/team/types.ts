// Team feature types. Mirror the backend DTOs produced by the
// organization handlers (Phase 1-3) and the shared session shape from
// useSession/useOrganization. Deliberately narrow — if a field exists
// on the backend but isn't consumed here, we don't import it.

export type OrgRole = "owner" | "admin" | "member" | "viewer"

export type TeamMember = {
  id: string
  organization_id: string
  user_id: string
  role: OrgRole
  title: string
  joined_at: string
  // The users table is joined server-side so the UI can render an
  // avatar + name without a second round-trip. Absent for legacy rows
  // before the backend added the join.
  user?: {
    id: string
    email: string
    display_name: string
    first_name: string
    last_name: string
  }
}

export type TeamInvitationStatus = "pending" | "accepted" | "cancelled" | "expired"

export type TeamInvitation = {
  id: string
  organization_id: string
  email: string
  first_name: string
  last_name: string
  title: string
  role: "admin" | "member" | "viewer"
  invited_by_user_id: string
  status: TeamInvitationStatus
  expires_at: string
  accepted_at?: string | null
  cancelled_at?: string | null
  created_at: string
  updated_at: string
}

// API envelopes — the backend wraps list endpoints with `data` +
// cursor pagination metadata. For the team lists we only care about
// the items themselves (org size is capped at ~100 in V1).
export type TeamMembersListResponse = {
  data: TeamMember[]
  next_cursor?: string
}

export type TeamInvitationsListResponse = {
  data: TeamInvitation[]
  next_cursor?: string
}

// Mutation payloads — shape matches the request DTOs on the backend.
export type SendInvitationPayload = {
  email: string
  first_name: string
  last_name: string
  title: string
  role: "admin" | "member" | "viewer"
}

export type UpdateMemberPayload = {
  role?: OrgRole
  title?: string
}

export type InitiateTransferPayload = {
  target_user_id: string
}

// Public invitation preview returned by GET /invitations/validate.
// Used by the email-link landing page to show the invitee who is
// inviting them, into which org, and as what role before they set
// a password. Does not include the token itself — the page has it
// in the URL.
export type InvitationPreview = {
  id: string
  organization_id: string
  organization_name: string
  organization_type: "agency" | "enterprise"
  email: string
  first_name: string
  last_name: string
  title: string
  role: "owner" | "admin" | "member" | "viewer"
  expires_at: string
}

export type AcceptInvitationPayload = {
  token: string
  password: string
}

// Role and permission catalogue surfaced by GET
// /api/v1/organizations/role-definitions. Used by the team page's
// "About roles" panel and the Edit Member modal's inline preview to
// show users what each role can actually do before they pick one.
//
// Backend returns English defaults for label/description; the
// frontend translates by key (`team.roles.<key>`,
// `team.permissionGroups.<group>`, etc.) and falls back to the
// English string when no translation exists.

export type RoleDefinitionPermission = {
  key: string
  group: string
  label: string
  description: string
  // Whether the Owner can toggle this permission through the
  // role-permissions editor. False means the permission is locked
  // forever (wallet.withdraw, org.delete, kyc.manage, …).
  //
  // Optional for backward compatibility: older test fixtures and
  // cached responses from before R17 shipped do not carry this
  // field — treat undefined as "overridable unknown, fall back to
  // backend truth" on the client side.
  overridable?: boolean
}

export type RoleDefinition = {
  key: OrgRole
  label: string
  description: string
  permissions: string[]
}

export type RoleDefinitionsResponse = {
  roles: RoleDefinition[]
  permissions: RoleDefinitionPermission[]
}

// ---------------------------------------------------------------------
// Role permissions editor (R17 — per-org customization)
// ---------------------------------------------------------------------

// The backend encodes the origin of each (role, perm) cell so the UI
// can render the right visual treatment: default states have no
// badge; granted/revoked overrides show a colored pill; locked cells
// show a lock icon and are not interactive.
export type RolePermissionCellState =
  | "default_granted"
  | "default_revoked"
  | "granted_override"
  | "revoked_override"
  | "locked"

// One (permission, state) pair as returned by the backend's
// /organizations/{id}/role-permissions endpoint.
export type RolePermissionCell = {
  key: string
  group: string
  label: string
  description: string
  granted: boolean
  state: RolePermissionCellState
  locked: boolean
}

// The full catalogue for a single role, ready to render as a column
// in the editor. `role === "owner"` rows are read-only — the backend
// marks every cell as locked.
export type RolePermissionsRow = {
  role: OrgRole
  label: string
  description: string
  permissions: RolePermissionCell[]
}

// Response of GET /organizations/{id}/role-permissions.
export type RolePermissionsMatrixResponse = {
  roles: RolePermissionsRow[]
}

// Payload of PATCH /organizations/{id}/role-permissions. The
// `overrides` map is the FULL desired state for the target role —
// any previous override not present here reverts to the default.
export type UpdateRolePermissionsPayload = {
  role: "admin" | "member" | "viewer"
  overrides: Record<string, boolean>
}

// Response of PATCH /organizations/{id}/role-permissions. Bundles
// the save summary and the refreshed matrix for a one-round-trip
// cache refresh.
export type UpdateRolePermissionsResponse = {
  role: OrgRole
  granted_keys: string[]
  revoked_keys: string[]
  affected_members: number
  matrix?: RolePermissionsMatrixResponse
}
