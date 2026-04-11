import { adminApi } from "@/shared/lib/api-client"
import type {
  AdminOrganizationDetail,
  AdminOrganizationMember,
  AdminUserListResponse,
  AdminUserResponse,
  UserFilters,
} from "../types"

export function listUsers(filters: UserFilters): Promise<AdminUserListResponse> {
  const params = new URLSearchParams()
  if (filters.role) params.set("role", filters.role)
  if (filters.status) params.set("status", filters.status)
  if (filters.search) params.set("search", filters.search)
  if (filters.page > 0) params.set("page", String(filters.page))
  if (filters.reported) params.set("reported", "true")
  params.set("limit", "20")

  const qs = params.toString()
  return adminApi<AdminUserListResponse>(`/api/v1/admin/users${qs ? `?${qs}` : ""}`)
}

export function getUser(id: string): Promise<AdminUserResponse> {
  return adminApi<AdminUserResponse>(`/api/v1/admin/users/${id}`)
}

export type SuspendUserPayload = {
  reason: string
  expires_at?: string
}

export type BanUserPayload = {
  reason: string
}

export function suspendUser(id: string, payload: SuspendUserPayload): Promise<void> {
  return adminApi(`/api/v1/admin/users/${id}/suspend`, {
    method: "POST",
    body: payload,
  })
}

export function unsuspendUser(id: string): Promise<void> {
  return adminApi(`/api/v1/admin/users/${id}/unsuspend`, {
    method: "POST",
    body: {},
  })
}

export function banUser(id: string, payload: BanUserPayload): Promise<void> {
  return adminApi(`/api/v1/admin/users/${id}/ban`, {
    method: "POST",
    body: payload,
  })
}

export function unbanUser(id: string): Promise<void> {
  return adminApi(`/api/v1/admin/users/${id}/unban`, {
    method: "POST",
    body: {},
  })
}

/* -------------------------------------------------------------------------- */
/* Phase 6 — Team management (admin override)                                 */
/* -------------------------------------------------------------------------- */

// Returns the team aggregate for a user (org + members + pending
// invitations + transfer state). Responds with 404 when the user has
// no organization — useful for distinguishing solo Providers from
// Owners before deciding whether to render the team section.
export function getUserOrganization(id: string): Promise<AdminOrganizationDetail> {
  return adminApi<AdminOrganizationDetail>(`/api/v1/admin/users/${id}/organization`)
}

export type ForceTransferPayload = {
  target_user_id: string
}

export function forceTransferOwnership(orgID: string, payload: ForceTransferPayload): Promise<void> {
  return adminApi(`/api/v1/admin/organizations/${orgID}/force-transfer`, {
    method: "POST",
    body: payload,
  })
}

export type ForceUpdateMemberRolePayload = {
  role: "admin" | "member" | "viewer"
}

export function forceUpdateMemberRole(
  orgID: string,
  userID: string,
  payload: ForceUpdateMemberRolePayload,
): Promise<AdminOrganizationMember> {
  return adminApi<AdminOrganizationMember>(
    `/api/v1/admin/organizations/${orgID}/members/${userID}`,
    { method: "PATCH", body: payload },
  )
}

export function forceRemoveMember(orgID: string, userID: string): Promise<void> {
  return adminApi(`/api/v1/admin/organizations/${orgID}/members/${userID}`, {
    method: "DELETE",
  })
}

export function forceCancelInvitation(orgID: string, invID: string): Promise<void> {
  return adminApi(`/api/v1/admin/organizations/${orgID}/invitations/${invID}`, {
    method: "DELETE",
  })
}
