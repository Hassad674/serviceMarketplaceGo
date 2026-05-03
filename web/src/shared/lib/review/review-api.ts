import { apiClient } from "@/shared/lib/api-client"
import type { Get, Post } from "@/shared/lib/api-paths"
import type { Review, AverageRating } from "@/shared/types/review"

// Re-export shared types for backward compatibility.
export type { Review, AverageRating }

export type ReviewListResponse = {
  data: Review[]
  next_cursor: string
  has_more: boolean
}

export type CanReviewResponse = {
  can_review: boolean
}

export type CreateReviewPayload = {
  proposal_id: string
  global_rating: number
  timeliness?: number
  communication?: number
  quality?: number
  comment?: string
  video_url?: string
  title_visible?: boolean
}

export async function fetchReviewsByUser(userId: string, cursor?: string) {
  const params = new URLSearchParams()
  if (cursor) params.set("cursor", cursor)
  const query = params.toString()
  const url = `/api/v1/reviews/user/${userId}${query ? `?${query}` : ""}`
  return apiClient<ReviewListResponse>(url)
}

export async function fetchAverageRating(userId: string) {
  return apiClient<Get<"/api/v1/reviews/average/{orgId}"> & { data: AverageRating }>(`/api/v1/reviews/average/${userId}`)
}

export async function fetchCanReview(proposalId: string) {
  return apiClient<Get<"/api/v1/reviews/can-review/{proposalId}"> & { data: CanReviewResponse }>(`/api/v1/reviews/can-review/${proposalId}`)
}

export async function createReview(payload: CreateReviewPayload) {
  return apiClient<Post<"/api/v1/reviews"> & { data: Review }>("/api/v1/reviews", {
    method: "POST",
    body: payload,
  })
}

import { API_BASE_URL } from "@/shared/lib/api-client"

const API_URL = API_BASE_URL

export async function uploadReviewVideo(file: File): Promise<string> {
  const formData = new FormData()
  formData.append("file", file)

  const res = await fetch(`${API_URL}/api/v1/upload/review-video`, {
    method: "POST",
    credentials: "include",
    body: formData,
  })

  if (!res.ok) {
    const err = await res.json().catch(() => ({ message: "Upload failed" }))
    throw new Error(err.message || "Upload failed")
  }

  const data = await res.json()
  return data.url
}
