export type ProposalStatus =
  | "pending"
  | "accepted"
  | "declined"
  | "withdrawn"
  | "paid"
  | "active"
  | "completion_requested"
  | "completed"
  | "disputed"

export type ProposalDocument = {
  id: string
  filename: string
  url: string
  size: number
  mime_type: string
}

// Phase 5: payment_mode is the UX hint for which form to render on
// the create page and which tracker variant to render on the detail
// page. The backend treats both modes identically — every proposal
// has at least one milestone, the flag only affects the UI.
export type PaymentMode = "one_time" | "milestone"

// Milestone status enum (mirrors backend internal/domain/milestone).
export type MilestoneStatus =
  | "pending_funding"
  | "funded"
  | "submitted"
  | "approved"
  | "released"
  | "disputed"
  | "cancelled"
  | "refunded"

// Per-milestone payload returned by the API alongside a proposal.
// Amount is in centimes (1 EUR = 100), same convention as the legacy
// proposal.amount field.
export type MilestoneResponse = {
  id: string
  sequence: number
  title: string
  description: string
  amount: number
  deadline?: string | null
  status: MilestoneStatus
  version: number
  funded_at?: string | null
  submitted_at?: string | null
  approved_at?: string | null
  released_at?: string | null
  disputed_at?: string | null
  cancelled_at?: string | null
}

export type ProposalResponse = {
  id: string
  conversation_id: string
  sender_id: string
  recipient_id: string
  title: string
  description: string
  amount: number
  deadline: string | null
  status: ProposalStatus
  parent_id: string | null
  version: number
  client_id: string
  provider_id: string
  client_name: string
  provider_name: string
  active_dispute_id: string | null
  // Most recent dispute ever opened on this proposal, regardless of its
  // current status. Set when a dispute is created, NEVER cleared, so the
  // project page can render the historical decision after restoration.
  last_dispute_id?: string | null
  documents: ProposalDocument[]
  // Phase 5: every proposal has at least one milestone (pre-phase-4
  // proposals were backfilled with a single synthetic one). The
  // current_milestone_sequence points to the milestone whose CTA
  // should be rendered on the detail page.
  payment_mode: PaymentMode
  milestones: MilestoneResponse[]
  current_milestone_sequence?: number
  accepted_at: string | null
  paid_at: string | null
  created_at: string
}

export type ProjectListResponse = {
  data: ProposalResponse[]
  next_cursor: string
  has_more: boolean
}

// Phase 10: the milestone editor uses string fields so the user can
// type freely without controlled-input fight; the create handler
// parses them into numbers + dates before sending to the API.
export type MilestoneFormItem = {
  title: string
  description: string
  amount: string
  deadline: string
}

export type ProposalFormData = {
  recipientId: string
  conversationId: string
  title: string
  description: string
  amount: string
  deadline: string
  files: File[]
  // Phase 10 additions:
  paymentMode: PaymentMode
  milestones: MilestoneFormItem[]
}

export type ProposalMessageMetadata = {
  proposal_id: string
  proposal_title: string
  proposal_amount: number
  proposal_status: ProposalStatus
  proposal_deadline: string | null
  proposal_sender_name: string
  proposal_documents_count: number
  proposal_version: number
  proposal_parent_id: string | null
  proposal_client_id: string
  proposal_provider_id: string
}

export type PaymentIntentResponse = {
  client_secret?: string
  payment_record_id?: string
  amounts?: {
    proposal_amount: number
    stripe_fee: number
    platform_fee: number
    client_total: number
    provider_payout: number
  }
  status?: string // "paid" for simulation mode
}

export type UploadURLResponse = {
  upload_url: string
  file_key: string
  public_url: string
}

export function createEmptyProposalForm(): ProposalFormData {
  return {
    recipientId: "",
    conversationId: "",
    title: "",
    description: "",
    amount: "",
    deadline: "",
    files: [],
    paymentMode: "one_time",
    milestones: [createEmptyMilestoneItem()],
  }
}

export function createEmptyMilestoneItem(): MilestoneFormItem {
  return {
    title: "",
    description: "",
    amount: "",
    deadline: "",
  }
}

// MAX_MILESTONES_PER_PROPOSAL mirrors the backend constant
// (MaxMilestonesPerProposal in internal/domain/milestone). Keep the
// two in sync manually — there is no shared schema.
export const MAX_MILESTONES_PER_PROPOSAL = 20

// sumMilestoneAmounts parses each form milestone amount and returns
// the running total in centimes. Invalid / empty entries contribute
// 0 so the sticky-footer "Total" reads cleanly while the user is
// still typing.
export function sumMilestoneAmounts(items: MilestoneFormItem[]): number {
  let total = 0
  for (const item of items) {
    const parsed = Number(item.amount)
    if (!Number.isNaN(parsed) && parsed > 0) {
      total += Math.round(parsed * 100)
    }
  }
  return total
}
