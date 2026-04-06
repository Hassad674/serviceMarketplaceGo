import { adminApi } from "@/shared/lib/api-client"
import type {
  AdminReviewListResponse,
  AdminReviewDetailResponse,
  ReviewFilters,
} from "../types"

export function listAdminReviews(filters: ReviewFilters): Promise<AdminReviewListResponse> {
  const params = new URLSearchParams()
  if (filters.search) params.set("search", filters.search)
  if (filters.rating) params.set("rating", filters.rating)
  if (filters.sort) params.set("sort", filters.sort)
  if (filters.filter) params.set("filter", filters.filter)
  if (filters.page > 0) params.set("page", String(filters.page))
  params.set("limit", "20")
  const qs = params.toString()
  return adminApi<AdminReviewListResponse>(`/api/v1/admin/reviews${qs ? `?${qs}` : ""}`)
}

export function getAdminReview(id: string): Promise<AdminReviewDetailResponse> {
  return adminApi<AdminReviewDetailResponse>(`/api/v1/admin/reviews/${id}`)
}

export function deleteAdminReview(id: string): Promise<void> {
  return adminApi(`/api/v1/admin/reviews/${id}`, { method: "DELETE" })
}
