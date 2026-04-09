export type DisputeStatus = "open" | "negotiation" | "escalated" | "resolved" | "cancelled"

export type AdminDispute = {
  id: string
  proposal_id: string
  conversation_id: string
  initiator_id: string
  respondent_id: string
  client_id: string
  provider_id: string
  reason: string
  description: string
  requested_amount: number
  proposal_amount: number
  status: DisputeStatus
  resolution_type: string | null
  resolution_amount_client: number | null
  resolution_amount_provider: number | null
  resolution_note: string | null
  initiator_role: "client" | "provider"
  ai_summary: string | null
  evidence: AdminDisputeEvidence[]
  counter_proposals: AdminCounterProposal[]
  escalated_at: string | null
  resolved_at: string | null
  created_at: string
}

export type AdminDisputeEvidence = {
  id: string
  filename: string
  url: string
  size: number
  mime_type: string
}

export type AdminCounterProposal = {
  id: string
  proposer_id: string
  amount_client: number
  amount_provider: number
  message: string
  status: string
  responded_at: string | null
  created_at: string
}

export type DisputeListResponse = {
  data: AdminDispute[]
  next_cursor: string
  has_more: boolean
}

export type DisputeCountResponse = {
  total: number
  open: number
  escalated: number
}

export type DisputeFilters = {
  status: string
  cursor: string
}
