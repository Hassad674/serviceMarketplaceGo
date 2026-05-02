import { apiClient } from "@/shared/lib/api-client"
import type {
  ProposalResponse,
  ProjectListResponse,
  UploadURLResponse,
  PaymentIntentResponse,
  PaymentMode,
} from "../types"

// MilestoneInputData is the per-milestone payload a milestone-mode
// CreateProposal call sends to the API. amount is in centimes.
// sequence MUST be consecutive starting at 1 — the backend domain
// rejects gaps and duplicates.
export type MilestoneInputData = {
  sequence: number
  title: string
  description: string
  amount: number
  deadline?: string
}

export type CreateProposalData = {
  recipient_id: string
  conversation_id: string
  title: string
  description: string
  amount: number
  deadline?: string
  documents?: { filename: string; url: string; size: number; mime_type: string }[]
  // Phase 5 additions:
  payment_mode?: PaymentMode
  milestones?: MilestoneInputData[]
}

export type ModifyProposalData = {
  title: string
  description: string
  amount: number
  deadline?: string
  documents?: { filename: string; url: string; size: number; mime_type: string }[]
  payment_mode?: PaymentMode
  milestones?: MilestoneInputData[]
}

export function createProposal(data: CreateProposalData): Promise<ProposalResponse> {
  return apiClient<ProposalResponse>("/api/v1/proposals", {
    method: "POST",
    body: data,
  })
}

export function getProposal(id: string): Promise<ProposalResponse> {
  return apiClient<ProposalResponse>(`/api/v1/proposals/${id}`)
}

// `acceptProposal` and `declineProposal` are shared with the
// `messaging` feature (P9). They live in
// `@/shared/lib/proposal/proposal-actions-api` and are re-exported
// here so existing intra-feature imports keep working.
export { acceptProposal, declineProposal } from "@/shared/lib/proposal/proposal-actions-api"

export function modifyProposal(id: string, data: ModifyProposalData): Promise<ProposalResponse> {
  return apiClient<ProposalResponse>(`/api/v1/proposals/${id}/modify`, {
    method: "POST",
    body: data,
  })
}

export function initiatePayment(id: string): Promise<PaymentIntentResponse> {
  return apiClient<PaymentIntentResponse>(`/api/v1/proposals/${id}/pay`, { method: "POST" })
}

export function confirmPayment(id: string): Promise<void> {
  return apiClient<void>(`/api/v1/proposals/${id}/confirm-payment`, { method: "POST" })
}

export function listProjects(cursor?: string): Promise<ProjectListResponse> {
  const params = cursor ? `?cursor=${encodeURIComponent(cursor)}` : ""
  return apiClient<ProjectListResponse>(`/api/v1/projects${params}`)
}

export function getUploadURL(filename: string, contentType: string): Promise<UploadURLResponse> {
  return apiClient<UploadURLResponse>("/api/v1/messaging/upload-url", {
    method: "POST",
    body: { filename, content_type: contentType },
  })
}

// Phase 5: milestone-explicit endpoints. The {mid} segment is
// validated against the proposal's current active milestone — a
// stale client view returns 409 Conflict so the frontend can refetch
// and retry on a fresh milestone id.
export function fundMilestone(
  proposalID: string,
  milestoneID: string,
): Promise<PaymentIntentResponse> {
  return apiClient<PaymentIntentResponse>(
    `/api/v1/proposals/${proposalID}/milestones/${milestoneID}/fund`,
    { method: "POST" },
  )
}

export function submitMilestone(proposalID: string, milestoneID: string): Promise<void> {
  return apiClient<void>(
    `/api/v1/proposals/${proposalID}/milestones/${milestoneID}/submit`,
    { method: "POST" },
  )
}

export function approveMilestone(proposalID: string, milestoneID: string): Promise<void> {
  return apiClient<void>(
    `/api/v1/proposals/${proposalID}/milestones/${milestoneID}/approve`,
    { method: "POST" },
  )
}

export function rejectMilestone(proposalID: string, milestoneID: string): Promise<void> {
  return apiClient<void>(
    `/api/v1/proposals/${proposalID}/milestones/${milestoneID}/reject`,
    { method: "POST" },
  )
}

export function cancelProposal(proposalID: string): Promise<void> {
  return apiClient<void>(`/api/v1/proposals/${proposalID}/cancel`, { method: "POST" })
}
