/**
 * Shared billing-profile types used across the `invoicing`, `wallet`,
 * and `subscription` features. The single source of truth for the
 * billing-profile data shape, lifted out of the invoicing feature so
 * other features can render the completion gate without importing
 * from `@/features/invoicing/...`.
 *
 * Types mirroring the backend invoicing DTOs (see
 * internal/handler/billing_profile_handler.go and invoice_handler.go).
 *
 * Kept intentionally narrow: every field that crosses the wire is
 * listed explicitly so a backend change that would silently widen
 * the contract still triggers a TypeScript error here. No `any`,
 * no opaque records.
 */

export type ProfileType = "individual" | "business"

/**
 * One field the backend considers missing for the billing profile to
 * be considered "complete". The reason is a machine-readable token
 * (e.g. "required", "invalid_format") that the UI maps to localized
 * copy — never displayed verbatim.
 */
export type MissingField = {
  field: string
  reason: string
}

export type BillingProfile = {
  organization_id: string
  profile_type: ProfileType
  legal_name: string
  trading_name: string
  legal_form: string
  tax_id: string
  vat_number: string
  /** ISO-8601 UTC, null when never validated against VIES. */
  vat_validated_at: string | null
  address_line1: string
  address_line2: string
  postal_code: string
  city: string
  country: string
  invoicing_email: string
  /** ISO-8601 UTC, null when never synced from Stripe KYC. */
  synced_from_kyc_at: string | null
}

export type BillingProfileSnapshot = {
  profile: BillingProfile
  missing_fields: MissingField[]
  is_complete: boolean
}

export type UpdateBillingProfileInput = {
  profile_type: ProfileType
  legal_name: string
  trading_name: string
  legal_form: string
  tax_id: string
  vat_number: string
  address_line1: string
  address_line2: string
  postal_code: string
  city: string
  country: string
  invoicing_email: string
}

export type VIESResult = {
  valid: boolean
  registered_name: string
  /** ISO-8601 UTC */
  checked_at: string
}

export type CurrentMonthLine = {
  milestone_id: string
  payment_record_id: string
  /** ISO-8601 UTC */
  released_at: string
  platform_fee_cents: number
  proposal_amount_cents: number
}

export type CurrentMonthAggregate = {
  /** ISO-8601 UTC */
  period_start: string
  /** ISO-8601 UTC */
  period_end: string
  milestone_count: number
  total_fee_cents: number
  lines: CurrentMonthLine[]
}
