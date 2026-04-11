import { apiClient } from "@/shared/lib/api-client"
import type {
  TeamMember,
  TeamMembersListResponse,
  TeamInvitation,
  TeamInvitationsListResponse,
  SendInvitationPayload,
  UpdateMemberPayload,
  InitiateTransferPayload,
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
