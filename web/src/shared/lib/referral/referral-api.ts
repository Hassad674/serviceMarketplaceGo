import { apiClient } from "@/shared/lib/api-client"

import type {
  Referral,
  RespondReferralInput,
} from "@/shared/types/referral"

const BASE = "/api/v1/referrals"

/**
 * Shared subset of the referral API: only the endpoints consumed by the
 * cross-feature `ReferralSystemMessage` / `ReferralActions` UX (P9 — the
 * messaging feature renders these inline in conversation timelines).
 *
 * The dashboard-only endpoints (listMyReferrals, listIncomingReferrals,
 * negotiations, attributions, commissions) stay in the referral feature.
 */

export async function getReferral(id: string): Promise<Referral> {
  return apiClient<Referral>(`${BASE}/${id}`)
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
