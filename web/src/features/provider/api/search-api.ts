import { apiClient } from "@/shared/lib/api-client"

// PublicProfileSummary describes the organization surfaced by
// marketplace search / directory pages: the team's display name,
// org type, photo, and review metrics.
export type PublicProfileSummary = {
  organization_id: string
  name: string
  org_type: string
  title: string
  photo_url: string
  referrer_enabled: boolean
  average_rating: number
  review_count: number
}

export type SearchType = "freelancer" | "agency" | "enterprise" | "referrer"

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
  orgId: string,
): Promise<PublicProfileSummary> {
  return apiClient<PublicProfileSummary>(`/api/v1/profiles/${orgId}`)
}
