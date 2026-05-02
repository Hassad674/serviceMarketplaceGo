// Domain types for the business-referral (apport d'affaires) feature.
//
// These mirror the backend response shape produced by
// internal/handler/dto/response/referral.go. They are hand-written rather
// than generated because the referral handler is not yet exposed in the
// project's OpenAPI schema — once it is, swap to the generated types and
// delete this file.

export type ReferralStatus =
  | "pending_provider"
  | "pending_referrer"
  | "pending_client"
  | "active"
  | "rejected"
  | "expired"
  | "cancelled"
  | "terminated"

export type ReferralActorRole = "referrer" | "provider" | "client"

export type ReferralNegotiationAction =
  | "proposed"
  | "countered"
  | "accepted"
  | "rejected"

// IntroSnapshot mirrors the backend struct of the same name. Both halves
// are optional because the apporteur picks which fields to reveal via
// per-field toggles in the creation wizard.
export type IntroSnapshot = {
  provider: ProviderSnapshot
  client: ClientSnapshot
}

export type ProviderSnapshot = {
  expertise_domains?: string[]
  years_experience?: number | null
  average_rating?: number | null
  review_count?: number | null
  pricing_min_cents?: number | null
  pricing_max_cents?: number | null
  pricing_currency?: string
  pricing_type?: string
  region?: string
  languages?: string[]
  availability_state?: string
}

export type ClientSnapshot = {
  industry?: string
  size_bucket?: string
  region?: string
  budget_estimate_min_cents?: number | null
  budget_estimate_max_cents?: number | null
  budget_currency?: string
  need_summary?: string
  timeline?: string
}

// Referral is the main aggregate the dashboard and detail pages render.
//
// IMPORTANT: rate_pct is OPTIONAL on purpose. The backend redacts it when
// the viewer is the client and the referral is in a pre-active state
// (Modèle A: the client never sees the commission rate). Components must
// handle the absent case gracefully — render "—" or hide the row.
export type Referral = {
  id: string
  referrer_id: string
  provider_id: string
  client_id: string
  rate_pct?: number
  duration_months: number
  status: ReferralStatus
  version: number
  intro_snapshot: IntroSnapshot
  intro_message_for_me?: string
  activated_at?: string
  expires_at?: string
  last_action_at: string
  rejection_reason?: string
  created_at: string
  updated_at: string
}

export type ReferralListResponse = {
  items: Referral[]
  next_cursor?: string
}

export type ReferralNegotiation = {
  id: string
  version: number
  actor_id: string
  actor_role: ReferralActorRole
  action: ReferralNegotiationAction
  rate_pct: number
  message: string
  created_at: string
}

// ReferralAttribution mirrors the backend DTO for GET /referrals/{id}/attributions.
// rate_pct_snapshot and ALL commission totals (paid, pending, escrow,
// clawed-back) are OMITTED for client viewers — the backend strips
// them (Modèle A). Components must handle undefined gracefully.
//
// milestones_total is the authoritative count (≥ 1 by domain rule);
// the UI renders "{paid}/{total}" from it. milestones_pending is kept
// for backwards compat but reflects only commissions already created —
// it is NOT total - paid.
//
// escrow_commission_cents previews the apporteur's share of funds
// currently held in escrow on funded-but-not-released milestones.
// Non-zero for in-progress missions before any Stripe transfer has
// fired; shown as "+ X € en séquestre" under the paid amount.
//
// clawed_back_commission_cents sums commissions that were paid then
// reversed after a dispute. Shown as "- X € reprises" when > 0.
export type ReferralAttribution = {
  id: string
  proposal_id: string
  proposal_title?: string
  proposal_status?: string
  rate_pct_snapshot?: number
  attributed_at: string
  total_commission_cents?: number
  pending_commission_cents?: number
  escrow_commission_cents?: number
  clawed_back_commission_cents?: number
  milestones_paid: number
  milestones_pending: number
  milestones_total: number
}

// ReferralCommission mirrors GET /referrals/{id}/commissions. Blocked
// for the client (403) so this type is only consumed by apporteur /
// provider views.
export type ReferralCommission = {
  id: string
  attribution_id: string
  milestone_id: string
  gross_amount_cents: number
  commission_cents: number
  currency: string
  status:
    | "pending"
    | "pending_kyc"
    | "paid"
    | "failed"
    | "cancelled"
    | "clawed_back"
  stripe_transfer_id?: string
  stripe_reversal_id?: string
  failure_reason?: string
  paid_at?: string
  clawed_back_at?: string
  created_at: string
}

// ─── Mutation payloads ────────────────────────────────────────────────────

export type SnapshotToggles = {
  include_expertise: boolean
  include_experience: boolean
  include_rating: boolean
  include_pricing: boolean
  include_region: boolean
  include_languages: boolean
  include_availability: boolean
}

export type CreateReferralInput = {
  provider_id: string
  client_id: string
  rate_pct: number
  duration_months: number
  intro_message_provider: string
  intro_message_client: string
  snapshot_toggles?: SnapshotToggles
}

// RespondInput is the unified payload posted to /respond. The backend
// dispatches to the right service method based on the JWT user role.
export type RespondAction =
  | "accept"
  | "reject"
  | "negotiate"
  | "cancel"
  | "terminate"

export type RespondReferralInput = {
  action: RespondAction
  new_rate_pct?: number
  message?: string
}

// ─── UI helpers ───────────────────────────────────────────────────────────

// statusTone groups statuses by visual category so badge components can
// pick a colour without a giant switch in every render.
export type StatusTone = "pending" | "active" | "terminal-success" | "terminal-failure"

export function statusTone(status: ReferralStatus): StatusTone {
  switch (status) {
    case "pending_provider":
    case "pending_referrer":
    case "pending_client":
      return "pending"
    case "active":
      return "active"
    case "rejected":
    case "expired":
    case "cancelled":
      return "terminal-failure"
    case "terminated":
      return "terminal-success"
  }
}

// formatRatePct renders the rate cleanly, falling back to a placeholder
// when the field is absent (client viewing pre-activation).
export function formatRatePct(rate: number | undefined): string {
  if (rate === undefined || rate === null) return "—"
  return `${rate.toFixed(rate % 1 === 0 ? 0 : 2)}%`
}
