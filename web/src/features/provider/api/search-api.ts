import { apiClient } from "@/shared/lib/api-client"

// PublicProfileSummarySkill matches the compact skill shape returned
// by the search endpoint — same contract as the full profile, kept
// locally so the search API stays independent of profile-api.ts.
export type PublicProfileSummarySkill = {
  skill_text: string
  display_text: string
}

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
  // Skills surfaced on the search result card. Backend always returns
  // an array (possibly empty) — never null. Older clients should treat
  // `undefined` as empty.
  skills?: PublicProfileSummarySkill[]
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
