// Types mirroring the backend invoicing DTOs (see
// internal/handler/billing_profile_handler.go and invoice_handler.go).
//
// Kept intentionally narrow: every field that crosses the wire is
// listed explicitly so a backend change that would silently widen
// the contract still triggers a TypeScript error here. No `any`,
// no opaque records.

export type ProfileType = "individual" | "business"

export type InvoiceSourceType =
  | "subscription"
  | "monthly_commission"
  | "credit_note"

// `MissingField` is shared with `wallet` and `subscription` (P9). Re-exported
// here so existing intra-feature imports keep working without churn.
import type { MissingField } from "@/shared/types/billing-profile"
export type { MissingField }

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

export type Invoice = {
  id: string
  number: string
  /** ISO-8601 UTC */
  issued_at: string
  source_type: InvoiceSourceType
  amount_incl_tax_cents: number
  currency: string
  /** Always empty in list responses — fetch the dedicated /pdf endpoint. */
  pdf_url: string
}

export type InvoicesPage = {
  data: Invoice[]
  next_cursor?: string
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
