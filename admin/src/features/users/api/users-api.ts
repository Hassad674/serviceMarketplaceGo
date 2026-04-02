import { adminApi } from "@/shared/lib/api-client"
import type { AdminUserListResponse, AdminUserResponse, UserFilters } from "../types"

export function listUsers(filters: UserFilters): Promise<AdminUserListResponse> {
  const params = new URLSearchParams()
  if (filters.role) params.set("role", filters.role)
  if (filters.status) params.set("status", filters.status)
  if (filters.search) params.set("search", filters.search)
  if (filters.cursor) params.set("cursor", filters.cursor)
  params.set("limit", "20")

  const qs = params.toString()
  return adminApi<AdminUserListResponse>(`/api/v1/admin/users${qs ? `?${qs}` : ""}`)
}

export function getUser(id: string): Promise<AdminUserResponse> {
  return adminApi<AdminUserResponse>(`/api/v1/admin/users/${id}`)
}
