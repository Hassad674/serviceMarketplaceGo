// Shared billing types (P9 — `FeePreview` UX is consumed cross-feature
// by the proposal creation / detail flows).
//
// The backend resolves the prestataire role (freelance / agency) from
// the JWT — never passed as a query parameter — and returns the full
// tier grid so the UI can render every bracket, highlight the active
// one, and show the exact fee applied to the requested amount.
//
// Amounts are in centimes (1 EUR = 100), matching the rest of the
// marketplace money-handling convention.

export type FeePreviewRole = "freelance" | "agency"

export type FeePreviewTier = {
  /** Human-readable label for the tier row (e.g. "200 € – 1 000 €"). */
  label: string
  /**
   * Inclusive upper bound of the tier in centimes.
   * `null` marks the open-ended top tier ("Plus de 1 000 €").
   */
  max_cents: number | null
  /** Flat fee applied when an amount falls inside this tier. */
  fee_cents: number
}

export type FeePreview = {
  amount_cents: number
  fee_cents: number
  net_cents: number
  role: FeePreviewRole
  /** Index into `tiers` marking the bracket the amount falls into. */
  active_tier_index: number
  tiers: FeePreviewTier[]
  /**
   * Whether the viewer (JWT caller) is the prestataire in the
   * (caller, recipient) pair. When `recipient_id` is omitted, the
   * backend assumes the caller is the provider and returns `true`
   * iff the caller's role is a provider role. When `recipient_id`
   * is supplied the backend runs `DetermineRoles` and fails closed
   * (`false`) on invalid combos — the UI uses that to hide the
   * preview from clients.
   */
  viewer_is_provider: boolean
  /**
   * Whether the viewer currently holds an active Premium subscription.
   * When `true`, the backend has already zeroed `fee_cents` and
   * `net_cents` equals `amount_cents`; the UI swaps the tier grid for a
   * subscription-active notice so the prestataire sees the waiver
   * unambiguously.
   */
  viewer_is_subscribed: boolean
}
