export type DisputeStatus = "open" | "negotiation" | "escalated" | "resolved" | "cancelled"

export type AdminDispute = {
  id: string
  proposal_id: string
  // Phase 8: every dispute is scoped to a single milestone. The
  // proposal_amount field actually carries the disputed milestone's
  // amount — the field name is preserved for backward compatibility
  // but the resolution split is on milestone.amount, not the
  // proposal total.
  milestone_id?: string
  milestone_sequence?: number
  milestone_title?: string
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
  ai_budget?: AIBudgetSummary
  ai_chat_history?: AIChatMessage[]
  evidence: AdminDisputeEvidence[]
  counter_proposals: AdminCounterProposal[]
  // Optional to tolerate older API responses; treat missing as no request.
  cancellation_requested_by?: string | null
  cancellation_requested_at?: string | null
  escalated_at: string | null
  resolved_at: string | null
  created_at: string
}

// AIChatMessage is one persisted turn of the admin AI chat for a dispute,
// loaded server-side with the rest of the dispute detail.
export type AIChatMessage = {
  id: string
  role: "user" | "assistant"
  content: string
  input_tokens: number
  output_tokens: number
  created_at: string
}

export type AIBudgetTier = "S" | "M" | "L" | "XL"

export type AIBudgetSummary = {
  tier: AIBudgetTier
  bonus_tokens: number
  summary_used_tokens: number
  summary_max_tokens: number
  chat_used_tokens: number
  chat_max_tokens: number
  total_used_tokens: number
  total_cost_eur: number
}

// AskAIResponse is the immediate response of the chat endpoint. The
// admin UI ignores its body in practice (the source of truth is the
// dispute refetch which contains the persisted full history), but it is
// kept here so the API function stays typed.
export type AskAIResponse = {
  answer: string
  input_tokens: number
  output_tokens: number
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
