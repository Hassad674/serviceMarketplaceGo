import { adminApi } from "@/shared/lib/api-client"
import type { AdminMediaListResponse, AdminMediaDetailResponse, MediaFilters } from "../types"

export function listAdminMedia(filters: MediaFilters): Promise<AdminMediaListResponse> {
  const params = new URLSearchParams()
  if (filters.status) params.set("status", filters.status)
  if (filters.type) params.set("type", filters.type)
  if (filters.context) params.set("context", filters.context)
  if (filters.search) params.set("search", filters.search)
  if (filters.sort) params.set("sort", filters.sort)
  if (filters.page > 0) params.set("page", String(filters.page))
  params.set("limit", "20")

  const qs = params.toString()
  return adminApi<AdminMediaListResponse>(`/api/v1/admin/media${qs ? `?${qs}` : ""}`)
}

export function getAdminMedia(id: string): Promise<AdminMediaDetailResponse> {
  return adminApi<AdminMediaDetailResponse>(`/api/v1/admin/media/${id}`)
}

export function approveMedia(id: string): Promise<void> {
  return adminApi(`/api/v1/admin/media/${id}/approve`, {
    method: "POST",
    body: {},
  })
}

export function rejectMedia(id: string): Promise<void> {
  return adminApi(`/api/v1/admin/media/${id}/reject`, {
    method: "POST",
    body: {},
  })
}

export function deleteMedia(id: string): Promise<void> {
  return adminApi(`/api/v1/admin/media/${id}`, {
    method: "DELETE",
  })
}
