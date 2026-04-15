import { apiClient } from "@/shared/lib/api-client"

import type {
  CreateReferralInput,
  Referral,
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
  return apiClient<ReferralListResponse>(`${BASE}/me${buildQuery(filter)}`)
}

// listIncomingReferrals fetches referrals where the current user is the
// provider party OR the client party. The backend merges both sides into
// a single response.
export async function listIncomingReferrals(
  filter: ListReferralsFilter = {},
): Promise<ReferralListResponse> {
  return apiClient<ReferralListResponse>(`${BASE}/incoming${buildQuery(filter)}`)
}

export async function getReferral(id: string): Promise<Referral> {
  return apiClient<Referral>(`${BASE}/${id}`)
}

export async function createReferral(
  input: CreateReferralInput,
): Promise<Referral> {
  return apiClient<Referral>(BASE, {
    method: "POST",
    body: input,
  })
}

export async function respondToReferral(
  id: string,
  input: RespondReferralInput,
): Promise<Referral> {
  return apiClient<Referral>(`${BASE}/${id}/respond`, {
    method: "POST",
    body: input,
  })
}

export async function listNegotiations(id: string): Promise<ReferralNegotiation[]> {
  return apiClient<ReferralNegotiation[]>(`${BASE}/${id}/negotiations`)
}
