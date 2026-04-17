import { apiClient } from "@/shared/lib/api-client"

// ProjectHistoryEntry is one attributed mission on the apporteur's
// reputation surface. Client identity is intentionally absent — B2B
// working relationships stay confidential, the surface only exposes
// the provider side.
export type ReferrerProjectHistoryEntry = {
  proposal_id: string
  proposal_title: string
  proposal_status: string
  provider_id: string
  provider_name: string
  rating: number | null
  comment: string
  reviewed_at: string | null
  completed_at: string | null
  attributed_at: string
}

// ReferrerReputation is the full reputation aggregate: a single rating
// stat pair computed across every reviewed, completed attribution,
// plus the cursor-paginated "projets apportés" history.
//
// rating_avg and review_count are summary stats returned once on the
// first page — they do NOT rotate as the caller pages through.
export type ReferrerReputation = {
  rating_avg: number
  review_count: number
  history: ReferrerProjectHistoryEntry[]
  next_cursor: string
  has_more: boolean
}

// getReferrerReputation loads the apporteur reputation aggregate.
// Keyed on orgID for URL symmetry with the rest of the referrer
// profile surface — the backend translates to the owner user_id
// because referrals reference users.
export async function getReferrerReputation(
  orgId: string,
  params: { cursor?: string; limit?: number } = {},
): Promise<ReferrerReputation> {
  const search = new URLSearchParams()
  if (params.cursor) search.set("cursor", params.cursor)
  if (params.limit) search.set("limit", String(params.limit))
  const qs = search.toString()
  const suffix = qs ? `?${qs}` : ""
  return apiClient<ReferrerReputation>(
    `/api/v1/referrer-profiles/${orgId}/reputation${suffix}`,
  )
}
