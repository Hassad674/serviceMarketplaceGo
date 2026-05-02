// Billing-profile types live in `@/shared/types/billing-profile` (P9 —
// the wallet feature renders the completion gate, and CurrentMonth is
// rendered above the wallet payout block). Invoice-only types stay
// scoped to this feature.

export type InvoiceSourceType =
  | "subscription"
  | "monthly_commission"
  | "credit_note"

// Re-export shared billing-profile types for back-compat.
import type {
  ProfileType,
  MissingField,
  BillingProfile,
  BillingProfileSnapshot,
  UpdateBillingProfileInput,
  VIESResult,
  CurrentMonthLine,
  CurrentMonthAggregate,
} from "@/shared/types/billing-profile"
export type {
  ProfileType,
  MissingField,
  BillingProfile,
  BillingProfileSnapshot,
  UpdateBillingProfileInput,
  VIESResult,
  CurrentMonthLine,
  CurrentMonthAggregate,
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
