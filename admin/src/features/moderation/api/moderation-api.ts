import { adminApi } from "@/shared/lib/api-client"
import type { ModerationListResponse, ModerationFilters } from "../types"

export function listModerationItems(filters: ModerationFilters): Promise<ModerationListResponse> {
  const params = new URLSearchParams()
  if (filters.source) params.set("source", filters.source)
  if (filters.type) params.set("type", filters.type)
  if (filters.status) params.set("status", filters.status)
  if (filters.sort) params.set("sort", filters.sort)
  if (filters.page > 0) params.set("page", String(filters.page))
  params.set("limit", "20")

  const qs = params.toString()
  return adminApi<ModerationListResponse>(`/api/v1/admin/moderation${qs ? `?${qs}` : ""}`)
}

// Action API calls -- each calls an existing backend endpoint directly.
// No cross-feature imports: these are pure functions using adminApi.

export function approveMedia(id: string): Promise<void> {
  return adminApi(`/api/v1/admin/media/${id}/approve`, { method: "POST", body: {} })
}

export function rejectMedia(id: string): Promise<void> {
  return adminApi(`/api/v1/admin/media/${id}/reject`, { method: "POST", body: {} })
}

export function deleteMedia(id: string): Promise<void> {
  return adminApi(`/api/v1/admin/media/${id}`, { method: "DELETE" })
}

export function approveMessageModeration(id: string): Promise<void> {
  return adminApi(`/api/v1/admin/messages/${id}/approve-moderation`, { method: "POST", body: {} })
}

export function hideMessage(id: string): Promise<void> {
  return adminApi(`/api/v1/admin/messages/${id}/hide`, { method: "POST", body: {} })
}

export function approveReviewModeration(id: string): Promise<void> {
  return adminApi(`/api/v1/admin/reviews/${id}/approve-moderation`, { method: "POST", body: {} })
}

export function deleteReview(id: string): Promise<void> {
  return adminApi(`/api/v1/admin/reviews/${id}`, { method: "DELETE" })
}

export function resolveReport(
  reportId: string,
  payload: { status: "resolved" | "dismissed"; admin_note: string },
): Promise<void> {
  return adminApi(`/api/v1/admin/reports/${reportId}/resolve`, {
    method: "POST",
    body: payload,
  })
}

export function restoreMessageModeration(id: string): Promise<void> {
  return adminApi(`/api/v1/admin/messages/${id}/restore-moderation`, { method: "POST", body: {} })
}

export function restoreReviewModeration(id: string): Promise<void> {
  return adminApi(`/api/v1/admin/reviews/${id}/restore-moderation`, { method: "POST", body: {} })
}

// Generic Phase 2 restore endpoint. Used for every content_type that
// is not "message" or "review" — profile_*, job_*, proposal_*,
// job_application_message, user_display_name. The backend route is
// POST /api/v1/admin/moderation/{content_type}/{content_id}/restore
// and writes the admin override into moderation_results without
// touching any source table.
export function restoreModerationGeneric(
  contentType: string,
  contentID: string,
): Promise<void> {
  return adminApi(
    `/api/v1/admin/moderation/${contentType}/${contentID}/restore`,
    { method: "POST", body: {} },
  )
}
