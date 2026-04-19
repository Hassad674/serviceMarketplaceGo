import { apiClient } from "@/shared/lib/api-client"
import type { FeePreview } from "../types"

/**
 * Fetches the platform fee preview for a given milestone amount.
 *
 * The prestataire's role (freelance / agency) is resolved server-side
 * from the JWT — NEVER pass role as a query parameter, as that would
 * let a client fake the cheaper grid. `amount_cents` is the only
 * input the endpoint accepts.
 */
export function getFeePreview(amountCents: number): Promise<FeePreview> {
  const safe = Math.max(0, Math.trunc(amountCents))
  return apiClient<FeePreview>(`/api/v1/billing/fee-preview?amount=${safe}`)
}
