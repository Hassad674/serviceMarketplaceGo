import { apiClient } from "@/shared/lib/api-client"

export type PublicProfileSummary = {
  user_id: string
  display_name: string
  first_name: string
  last_name: string
  role: string
  title: string
  photo_url: string
  referrer_enabled: boolean
  average_rating: number
  review_count: number
}

export type SearchType = "freelancer" | "agency" | "referrer"

export type SearchResponse = {
  data: PublicProfileSummary[]
  next_cursor: string
  has_more: boolean
}

export async function searchProfiles(
  type: SearchType,
  cursor?: string,
): Promise<SearchResponse> {
  const params = new URLSearchParams({ type })
  if (cursor) params.set("cursor", cursor)
  return apiClient<SearchResponse>(
    `/api/v1/profiles/search?${params.toString()}`,
  )
}

export async function getPublicProfile(
  userId: string,
): Promise<PublicProfileSummary> {
  return apiClient<PublicProfileSummary>(`/api/v1/profiles/${userId}`)
}
