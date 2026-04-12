import { apiClient } from "@/shared/lib/api-client"
import type {
  TeamMember,
  TeamMembersListResponse,
  TeamInvitation,
  TeamInvitationsListResponse,
  SendInvitationPayload,
  UpdateMemberPayload,
  InitiateTransferPayload,
  InvitationPreview,
  AcceptInvitationPayload,
  RoleDefinitionsResponse,
  RolePermissionsMatrixResponse,
  UpdateRolePermissionsPayload,
  UpdateRolePermissionsResponse,
} from "../types"

// Pure async functions that talk to the organization endpoints
// shipped in Phases 1-3. Each function maps 1:1 to a backend route so
// the hook layer above stays a dumb TanStack Query wrapper.

/* ------------------------------------------------------------------ */
/* Members                                                            */
/* ------------------------------------------------------------------ */

export function listMembers(orgID: string): Promise<TeamMembersListResponse> {
  return apiClient<TeamMembersListResponse>(
    `/api/v1/organizations/${orgID}/members?limit=100`,
  )
}

export function updateMember(
  orgID: string,
  userID: string,
  payload: UpdateMemberPayload,
): Promise<TeamMember> {
  return apiClient<TeamMember>(
    `/api/v1/organizations/${orgID}/members/${userID}`,
    { method: "PATCH", body: payload },
  )
}

export function removeMember(orgID: string, userID: string): Promise<void> {
  return apiClient<void>(
    `/api/v1/organizations/${orgID}/members/${userID}`,
    { method: "DELETE" },
  )
}

export function leaveOrganization(orgID: string): Promise<void> {
  return apiClient<void>(
    `/api/v1/organizations/${orgID}/leave`,
    { method: "POST", body: {} },
  )
}

/* ------------------------------------------------------------------ */
/* Role definitions (R13)                                              */
/* ------------------------------------------------------------------ */

// Static catalogue of roles and permissions surfaced by the backend.
// Used by the team page's "About roles" panel and the Edit Member
// modal's inline preview. Cached aggressively because the catalogue
// only changes when the backend deploys a new permission constant.
export function getRoleDefinitions(): Promise<RoleDefinitionsResponse> {
  return apiClient<RoleDefinitionsResponse>(
    `/api/v1/organizations/role-definitions`,
  )
}

/* ------------------------------------------------------------------ */
/* Role permissions editor (R17 — per-org customization)               */
/* ------------------------------------------------------------------ */

// Returns the full customized permission matrix for an organization.
// Every role row (Owner, Admin, Member, Viewer) is included, with
// cells pre-resolved into default / granted_override / revoked_override
// / locked states ready for direct rendering.
export function getRolePermissionsMatrix(
  orgID: string,
): Promise<RolePermissionsMatrixResponse> {
  return apiClient<RolePermissionsMatrixResponse>(
    `/api/v1/organizations/${orgID}/role-permissions`,
  )
}

// Saves a role's full override map. The backend replaces the
// previous state for that role in one shot — any cell missing from
// `overrides` reverts to the default. Only the Owner can call this
// endpoint; Admins receive a 403.
export function updateRolePermissions(
  orgID: string,
  payload: UpdateRolePermissionsPayload,
): Promise<UpdateRolePermissionsResponse> {
  return apiClient<UpdateRolePermissionsResponse>(
    `/api/v1/organizations/${orgID}/role-permissions`,
    { method: "PATCH", body: payload },
  )
}

/* ------------------------------------------------------------------ */
/* Invitations                                                        */
/* ------------------------------------------------------------------ */

export function listInvitations(orgID: string): Promise<TeamInvitationsListResponse> {
  // Phase 2 backend list endpoint returns all non-terminal invitations
  // by default; filter to pending client-side if needed. Cursor is
  // optional — V1 caps are low enough to fit in a single page.
  return apiClient<TeamInvitationsListResponse>(
    `/api/v1/organizations/${orgID}/invitations?limit=100`,
  )
}

export function sendInvitation(
  orgID: string,
  payload: SendInvitationPayload,
): Promise<TeamInvitation> {
  return apiClient<TeamInvitation>(
    `/api/v1/organizations/${orgID}/invitations`,
    { method: "POST", body: payload },
  )
}

export function resendInvitation(orgID: string, invitationID: string): Promise<TeamInvitation> {
  return apiClient<TeamInvitation>(
    `/api/v1/organizations/${orgID}/invitations/${invitationID}/resend`,
    { method: "POST", body: {} },
  )
}

export function cancelInvitation(orgID: string, invitationID: string): Promise<void> {
  return apiClient<void>(
    `/api/v1/organizations/${orgID}/invitations/${invitationID}`,
    { method: "DELETE" },
  )
}

/* ------------------------------------------------------------------ */
/* Ownership transfer                                                 */
/* ------------------------------------------------------------------ */

export function initiateTransferOwnership(
  orgID: string,
  payload: InitiateTransferPayload,
): Promise<void> {
  return apiClient<void>(
    `/api/v1/organizations/${orgID}/transfer`,
    { method: "POST", body: payload },
  )
}

export function cancelTransferOwnership(orgID: string): Promise<void> {
  return apiClient<void>(
    `/api/v1/organizations/${orgID}/transfer`,
    { method: "DELETE" },
  )
}

export function acceptTransferOwnership(orgID: string): Promise<void> {
  return apiClient<void>(
    `/api/v1/organizations/${orgID}/transfer/accept`,
    { method: "POST", body: {} },
  )
}

export function declineTransferOwnership(orgID: string): Promise<void> {
  return apiClient<void>(
    `/api/v1/organizations/${orgID}/transfer/decline`,
    { method: "POST", body: {} },
  )
}

/* ------------------------------------------------------------------ */
/* Public invitation landing (email link)                              */
/* ------------------------------------------------------------------ */

// The two endpoints below are the ONLY team endpoints that are
// public (no auth cookie). They power the /invitation/[token] email
// landing page — the invitee has no account yet, so the regular
// authenticated flow doesn't apply.

export function validateInvitation(token: string): Promise<InvitationPreview> {
  return apiClient<InvitationPreview>(
    `/api/v1/invitations/validate?token=${encodeURIComponent(token)}`,
  )
}

export function acceptInvitation(payload: AcceptInvitationPayload): Promise<unknown> {
  // The backend returns a full auth envelope on success and sets the
  // session cookie; the caller just needs to hard-redirect so the
  // whole React tree + TanStack cache is re-initialised with the new
  // session. We don't need to consume the return value.
  return apiClient<unknown>(`/api/v1/invitations/accept`, {
    method: "POST",
    body: payload,
    headers: { "X-Auth-Mode": "cookie" },
  })
}
