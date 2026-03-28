import { apiClient } from "@/shared/lib/api-client"

export type Review = {
  id: string
  proposal_id: string
  reviewer_id: string
  reviewed_id: string
  global_rating: number
  timeliness: number | null
  communication: number | null
  quality: number | null
  comment: string
  video_url: string | null
  created_at: string
}

export type ReviewListResponse = {
  data: Review[]
  next_cursor: string
  has_more: boolean
}

export type AverageRating = {
  average: number
  count: number
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
}

export async function fetchReviewsByUser(userId: string, cursor?: string) {
  const params = new URLSearchParams()
  if (cursor) params.set("cursor", cursor)
  const query = params.toString()
  const url = `/api/v1/reviews/user/${userId}${query ? `?${query}` : ""}`
  return apiClient<ReviewListResponse>(url)
}

export async function fetchAverageRating(userId: string) {
  return apiClient<{ data: AverageRating }>(`/api/v1/reviews/average/${userId}`)
}

export async function fetchCanReview(proposalId: string) {
  return apiClient<{ data: CanReviewResponse }>(`/api/v1/reviews/can-review/${proposalId}`)
}

export async function createReview(payload: CreateReviewPayload) {
  return apiClient<{ data: Review }>("/api/v1/reviews", {
    method: "POST",
    body: payload,
  })
}

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8083"

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
