import { apiClient } from "@/shared/lib/api-client"
import type { DisputeResponse } from "../types"

export type OpenDisputeData = {
  proposal_id: string
  reason: string
  description: string
  message_to_party: string
  requested_amount: number
  attachments?: { filename: string; url: string; size: number; mime_type: string }[]
}

export type CounterProposeData = {
  amount_client: number
  amount_provider: number
  message?: string
  attachments?: { filename: string; url: string; size: number; mime_type: string }[]
}

export function openDispute(data: OpenDisputeData): Promise<DisputeResponse> {
  return apiClient<DisputeResponse>("/api/v1/disputes", {
    method: "POST",
    body: data,
  })
}

export function getDispute(id: string): Promise<DisputeResponse> {
  return apiClient<DisputeResponse>(`/api/v1/disputes/${id}`)
}

export function counterPropose(disputeId: string, data: CounterProposeData): Promise<{ id: string }> {
  return apiClient<{ id: string }>(`/api/v1/disputes/${disputeId}/counter-propose`, {
    method: "POST",
    body: data,
  })
}

export function respondToCounter(disputeId: string, cpId: string, accept: boolean): Promise<{ status: string }> {
  return apiClient<{ status: string }>(`/api/v1/disputes/${disputeId}/counter-proposals/${cpId}/respond`, {
    method: "POST",
    body: { accept },
  })
}

export type CancelDisputeResult = { status: "cancelled" | "cancellation_requested" }

export function cancelDispute(id: string): Promise<CancelDisputeResult> {
  return apiClient<CancelDisputeResult>(`/api/v1/disputes/${id}/cancel`, {
    method: "POST",
  })
}

export function respondToCancellation(
  id: string,
  accept: boolean,
): Promise<{ status: "cancelled" | "refused" }> {
  return apiClient<{ status: "cancelled" | "refused" }>(
    `/api/v1/disputes/${id}/cancellation/respond`,
    {
      method: "POST",
      body: { accept },
    },
  )
}
