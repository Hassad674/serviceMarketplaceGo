import { apiClient } from "@/shared/lib/api-client"
import type { ProposalResponse, ProjectListResponse } from "../types"

export type CreateProposalData = {
  recipient_id: string
  conversation_id: string
  title: string
  description: string
  amount: number
  deadline?: string
  documents?: { filename: string; url: string; size: number; mime_type: string }[]
}

export type ModifyProposalData = {
  title: string
  description: string
  amount: number
  deadline?: string
  documents?: { filename: string; url: string; size: number; mime_type: string }[]
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

export function acceptProposal(id: string): Promise<void> {
  return apiClient<void>(`/api/v1/proposals/${id}/accept`, { method: "POST" })
}

export function declineProposal(id: string): Promise<void> {
  return apiClient<void>(`/api/v1/proposals/${id}/decline`, { method: "POST" })
}

export function modifyProposal(id: string, data: ModifyProposalData): Promise<ProposalResponse> {
  return apiClient<ProposalResponse>(`/api/v1/proposals/${id}/modify`, {
    method: "POST",
    body: data,
  })
}

export function simulatePayment(id: string): Promise<void> {
  return apiClient<void>(`/api/v1/proposals/${id}/pay`, { method: "POST" })
}

export function requestCompletion(id: string): Promise<void> {
  return apiClient<void>(`/api/v1/proposals/${id}/request-completion`, { method: "POST" })
}

export function completeProposal(id: string): Promise<void> {
  return apiClient<void>(`/api/v1/proposals/${id}/complete`, { method: "POST" })
}

export function rejectCompletion(id: string): Promise<void> {
  return apiClient<void>(`/api/v1/proposals/${id}/reject-completion`, { method: "POST" })
}

export function listProjects(cursor?: string): Promise<ProjectListResponse> {
  const params = cursor ? `?cursor=${encodeURIComponent(cursor)}` : ""
  return apiClient<ProjectListResponse>(`/api/v1/projects${params}`)
}
