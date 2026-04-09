export type DisputeStatus = "open" | "negotiation" | "escalated" | "resolved" | "cancelled"

export type DisputeReason =
  | "work_not_conforming"
  | "non_delivery"
  | "insufficient_quality"
  | "client_ghosting"
  | "scope_creep"
  | "refusal_to_validate"
  | "harassment"
  | "other"

export type EvidenceResponse = {
  id: string
  filename: string
  url: string
  size: number
  mime_type: string
}

export type CounterProposalResponse = {
  id: string
  proposer_id: string
  amount_client: number
  amount_provider: number
  message: string
  status: "pending" | "accepted" | "rejected" | "superseded"
  responded_at: string | null
  created_at: string
}

export type DisputeResponse = {
  id: string
  proposal_id: string
  conversation_id: string
  initiator_id: string
  respondent_id: string
  client_id: string
  provider_id: string
  reason: DisputeReason
  description: string
  requested_amount: number
  proposal_amount: number
  status: DisputeStatus
  resolution_type: string | null
  resolution_amount_client: number | null
  resolution_amount_provider: number | null
  resolution_note: string | null
  initiator_role: "client" | "provider"
  evidence: EvidenceResponse[]
  counter_proposals: CounterProposalResponse[]
  // Optional to tolerate older API responses that don't include these fields.
  // Treat missing as "no cancellation request pending".
  cancellation_requested_by?: string | null
  cancellation_requested_at?: string | null
  escalated_at: string | null
  resolved_at: string | null
  created_at: string
}

export type DisputeListResponse = {
  data: DisputeResponse[]
  next_cursor: string
  has_more: boolean
}
