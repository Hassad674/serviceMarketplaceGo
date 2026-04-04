import { adminApi } from "@/shared/lib/api-client"
import type { AdminUserListResponse, AdminUserResponse, UserFilters } from "../types"

export function listUsers(filters: UserFilters): Promise<AdminUserListResponse> {
  const params = new URLSearchParams()
  if (filters.role) params.set("role", filters.role)
  if (filters.status) params.set("status", filters.status)
  if (filters.search) params.set("search", filters.search)
  if (filters.cursor) params.set("cursor", filters.cursor)
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
