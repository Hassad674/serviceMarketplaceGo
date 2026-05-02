import { apiClient } from "@/shared/lib/api-client"

/**
 * Shared proposal action endpoints. Lifted out of
 * `features/proposal/api/proposal-api` so the messaging feature can
 * render accept/decline buttons inside a proposal card without
 * importing from the proposal feature directly. The proposal
 * feature also imports these from here (single source of truth).
 */

/** POST /api/v1/proposals/:id/accept */
export function acceptProposal(id: string): Promise<void> {
  return apiClient<void>(`/api/v1/proposals/${id}/accept`, { method: "POST" })
}

/** POST /api/v1/proposals/:id/decline */
export function declineProposal(id: string): Promise<void> {
  return apiClient<void>(`/api/v1/proposals/${id}/decline`, { method: "POST" })
}
