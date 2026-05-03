import { apiClient } from "@/shared/lib/api-client"

import type { Get } from "@/shared/lib/api-paths"
// Tier 1 taxonomy duplicated here to keep the search API self-
// contained — it intentionally does NOT import from any feature's
// profile-api so /search and the directory pages can render without
// pulling the full profile module into the bundle.
export type SearchWorkMode = "remote" | "on_site" | "hybrid"

export type SearchAvailabilityStatus =
  | "available_now"
  | "available_soon"
  | "not_available"

export type SearchPricingKind = "direct" | "referral"

export type SearchPricingType =
  | "daily"
  | "hourly"
  | "project_from"
  | "project_range"
  | "commission_pct"
  | "commission_flat"

// SearchPricing matches the backend Pricing row, identical to the full
// profile's Pricing type. Kept local on purpose (see header comment).
export type SearchPricing = {
  kind: SearchPricingKind
  type: SearchPricingType
  min_amount: number
  max_amount: number | null
  currency: string
  note: string
  negotiable: boolean
}

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
  // owner_user_id is the user at the top of the org. The business-
  // referral feature uses this as the "party id" when the apporteur
  // picks a provider from the search results, since the referral
  // backend indexes referrals on users, not orgs.
  owner_user_id: string
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
  // --- Tier 1 signals surfaced on the listing card. All fields are
  // optional because older orgs may not have completed Tier 1 yet.
  city?: string
  country_code?: string
  work_mode?: SearchWorkMode[]
  languages_professional?: string[]
  availability_status?: SearchAvailabilityStatus
  pricing?: SearchPricing[]
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
  return apiClient<Get<"/api/v1/profiles/search"> & SearchResponse>(
    `/api/v1/profiles/search?${params.toString()}`,
  )
}

export async function getPublicProfile(
  orgId: string,
): Promise<PublicProfileSummary> {
  return apiClient<Get<"/api/v1/profiles/{orgId}"> & PublicProfileSummary>(`/api/v1/profiles/${orgId}`)
}
