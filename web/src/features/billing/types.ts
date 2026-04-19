// Types returned by GET /api/v1/billing/fee-preview.
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
}
