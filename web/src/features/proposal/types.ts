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

// MilestoneDeadlineErrorKey is the i18n key for an inline date-row error.
// "not_after_previous" — milestone N+1 deadline is not strictly after N's.
// "after_project_deadline" — milestone deadline exceeds the project deadline.
// Returned per row index so the form can render the message under the
// offending date input only (instead of a single global banner).
export type MilestoneDeadlineErrorKey =
  | "not_after_previous"
  | "after_project_deadline"

// validateMilestoneDeadlines mirrors the backend domain rule:
// milestone N+1's deadline must be STRICTLY after milestone N's
// (same-day rejected). Milestones without a deadline are skipped —
// the deadline is optional and only ordered milestones are checked.
//
// Optionally enforces a project-level upper bound: if projectDeadline
// is provided, every set milestone deadline must be ≤ projectDeadline
// (equality allowed: the project deadline IS the natural last day).
//
// Returns a sparse map keyed by row index — only rows that violate a
// rule appear, so the editor can render one message per offending row.
// An empty map means "all rows are valid".
//
// All inputs are YYYY-MM-DD strings (the format the date picker emits).
// Empty strings (`""`) mean "no deadline" and are skipped.
export function validateMilestoneDeadlines(
  milestones: MilestoneFormItem[],
  projectDeadline?: string,
): Record<number, MilestoneDeadlineErrorKey> {
  const errors: Record<number, MilestoneDeadlineErrorKey> = {}

  // Walk in order and track the last NON-NIL deadline so we can compare
  // each new one against it. Sparse deadlines (a row with no date in
  // the middle) skip the check for that row but the next set deadline
  // still has to come after the LAST set deadline.
  let lastSetDeadline: string | undefined
  for (let i = 0; i < milestones.length; i++) {
    const current = milestones[i].deadline
    if (!current) continue
    if (lastSetDeadline && current <= lastSetDeadline) {
      // YYYY-MM-DD string compare = chronological compare.
      // <= covers BOTH equal-date AND earlier-date — both invalid
      // under the strict-after contract.
      errors[i] = "not_after_previous"
    } else if (projectDeadline && current > projectDeadline) {
      errors[i] = "after_project_deadline"
    }
    lastSetDeadline = current
  }
  return errors
}

// minDateForMilestone returns the YYYY-MM-DD string the date picker
// for milestone at `index` should refuse to go below. Computed by
// finding the latest deadline among the previous milestones (strictly
// before `index`) and adding one calendar day, falling back to today
// when no previous deadline is set. The returned string is always a
// valid YYYY-MM-DD usable directly as the picker's `min` attribute.
//
// We add ONE day (not zero) to mirror the backend's strict-after
// rule — letting the picker land on the same day would silently
// accept a value that the form-level validator (and the API) will
// later reject.
export function minDateForMilestone(
  milestones: MilestoneFormItem[],
  index: number,
  todayIso: string,
): string {
  let latestPrev: string | undefined
  for (let i = 0; i < index; i++) {
    const d = milestones[i].deadline
    if (d && (!latestPrev || d > latestPrev)) {
      latestPrev = d
    }
  }
  if (!latestPrev) return todayIso
  const next = addOneDay(latestPrev)
  // Final picker bound is the LATER of (today, prev+1) so the user
  // can never select a past date even if a previous milestone had a
  // historical deadline (legacy data, or a buggy resync).
  return next > todayIso ? next : todayIso
}

// addOneDay shifts a YYYY-MM-DD string by exactly one calendar day.
// Returns the result as YYYY-MM-DD. Uses UTC arithmetic to dodge
// daylight-saving-time edge cases that would otherwise drift the
// boundary by an hour twice a year.
function addOneDay(ymd: string): string {
  const [yStr, mStr, dStr] = ymd.split("-")
  const date = new Date(Date.UTC(Number(yStr), Number(mStr) - 1, Number(dStr)))
  date.setUTCDate(date.getUTCDate() + 1)
  const y = date.getUTCFullYear()
  const m = String(date.getUTCMonth() + 1).padStart(2, "0")
  const d = String(date.getUTCDate()).padStart(2, "0")
  return `${y}-${m}-${d}`
}
