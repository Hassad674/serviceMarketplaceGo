import { apiClient } from "@/shared/lib/api-client"
import type { Get, Post } from "@/shared/lib/api-paths"

import type {
  CreateReferralInput,
  Referral,
  ReferralAttribution,
  ReferralCommission,
  ReferralListResponse,
  ReferralNegotiation,
  ReferralStatus,
  RespondReferralInput,
} from "../types"

const BASE = "/api/v1/referrals"

// listMyReferrals fetches the dashboard list — referrals where the current
// user is the apporteur. Optional status filter.
export type ListReferralsFilter = {
  statuses?: ReferralStatus[]
  cursor?: string
}

function buildQuery(filter: ListReferralsFilter): string {
  const params = new URLSearchParams()
  if (filter.statuses) {
    for (const s of filter.statuses) params.append("status", s)
  }
  if (filter.cursor) params.set("cursor", filter.cursor)
  const qs = params.toString()
  return qs ? `?${qs}` : ""
}

export async function listMyReferrals(
  filter: ListReferralsFilter = {},
): Promise<ReferralListResponse> {
  return apiClient<Get<"/api/v1/referrals/me"> & ReferralListResponse>(
    `${BASE}/me${buildQuery(filter)}`,
  )
}

// listIncomingReferrals fetches referrals where the current user is the
// provider party OR the client party. The backend merges both sides into
// a single response.
export async function listIncomingReferrals(
  filter: ListReferralsFilter = {},
): Promise<ReferralListResponse> {
  return apiClient<Get<"/api/v1/referrals/incoming"> & ReferralListResponse>(
    `${BASE}/incoming${buildQuery(filter)}`,
  )
}

// `getReferral` and `respondToReferral` are shared with the messaging
// feature (P9 — `ReferralSystemMessage` is rendered inline in
// conversation timelines). They live in
// `@/shared/lib/referral/referral-api` and are re-exported here so
// existing intra-feature imports keep working.
export {
  getReferral,
  respondToReferral,
} from "@/shared/lib/referral/referral-api"

export async function createReferral(
  input: CreateReferralInput,
): Promise<Referral> {
  return apiClient<Post<"/api/v1/referrals"> & Referral>(BASE, {
    method: "POST",
    body: input,
  })
}

export async function listNegotiations(id: string): Promise<ReferralNegotiation[]> {
  return apiClient<Get<"/api/v1/referrals/{id}/negotiations"> & ReferralNegotiation[]>(
    `${BASE}/${id}/negotiations`,
  )
}

// listAttributions returns the proposals attributed during the
// exclusivity window, enriched with proposal title + status + aggregate
// commission stats. Commission amounts are absent for client viewers —
// Modèle A confidentiality.
export async function listAttributions(id: string): Promise<ReferralAttribution[]> {
  return apiClient<Get<"/api/v1/referrals/{id}/attributions"> & ReferralAttribution[]>(
    `${BASE}/${id}/attributions`,
  )
}

// listCommissions returns every commission row for a referral across
// all attributions. Reserved for apporteur + provider parties — the
// client receives 403 from this endpoint.
export async function listCommissions(id: string): Promise<ReferralCommission[]> {
  return apiClient<Get<"/api/v1/referrals/{id}/commissions"> & ReferralCommission[]>(
    `${BASE}/${id}/commissions`,
  )
}
