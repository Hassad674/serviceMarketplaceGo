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
