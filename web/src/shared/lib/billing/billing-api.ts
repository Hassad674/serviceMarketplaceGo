import { apiClient } from "@/shared/lib/api-client"
import type { Get } from "@/shared/lib/api-paths"
import type { FeePreview } from "@/shared/types/billing"

/**
 * Shared `getFeePreview` (P9 — consumed cross-feature by the proposal
 * creation / detail flows). Lifted from `features/billing/api/billing-api`.
 *
 * The prestataire's role (freelance / agency) is resolved server-side
 * from the JWT — NEVER pass role as a query parameter, as that would
 * let a client fake the cheaper grid. `amount_cents` is the only
 * numeric input the endpoint accepts.
 *
 * When `recipientId` is provided, the backend runs the same role
 * resolution as the proposal creation endpoint (`DetermineRoles`) and
 * sets `viewer_is_provider` in the response, so the UI can hide the
 * preview for client-side viewers (enterprise, agency paired with a
 * provider, etc.).
 */
export function getFeePreview(
  amountCents: number,
  recipientId?: string,
): Promise<FeePreview> {
  const safe = Math.max(0, Math.trunc(amountCents))
  const params = new URLSearchParams({ amount: String(safe) })
  if (recipientId) params.set("recipient_id", recipientId)
  return apiClient<Get<"/api/v1/billing/fee-preview"> & FeePreview>(`/api/v1/billing/fee-preview?${params.toString()}`)
}
