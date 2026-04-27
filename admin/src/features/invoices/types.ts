// Types for the admin "all invoices ever emitted" listing page.
//
// The backend returns a unified row that collapses invoices and credit
// notes onto a single shape — see backend/internal/handler/admin_invoice_handler.go.

export type AdminInvoiceRow = {
  id: string
  number: string
  is_credit_note: boolean
  recipient_org_id: string
  recipient_legal_name: string
  issued_at: string
  amount_incl_tax_cents: number
  currency: string
  tax_regime: string
  status: string
  original_invoice_id?: string | null
  source_type?: string
}

export type AdminInvoiceListResponse = {
  data: AdminInvoiceRow[]
  next_cursor?: string
  has_more: boolean
}

// AdminInvoiceTypeFilter mirrors the backend's accepted "status" filter
// values plus the empty "all" sentinel. Renamed to "type" on the UI
// because the user thinks in terms of "what kind of document is this"
// — invoice subscription, monthly commission, or credit note.
export type AdminInvoiceTypeFilter =
  | ""
  | "subscription"
  | "monthly_commission"
  | "credit_note"

export type AdminInvoiceFilters = {
  recipient_org_id: string
  status: AdminInvoiceTypeFilter
  date_from: string
  date_to: string
  min_amount_cents: string
  max_amount_cents: string
  search: string
  cursor: string
}

export const EMPTY_ADMIN_INVOICE_FILTERS: AdminInvoiceFilters = {
  recipient_org_id: "",
  status: "",
  date_from: "",
  date_to: "",
  min_amount_cents: "",
  max_amount_cents: "",
  search: "",
  cursor: "",
}
