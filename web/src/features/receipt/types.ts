/**
 * Transaction-receipt types matching the backend DTO shape exposed
 * by `GET /api/v1/receipts*`.
 *
 * Important — receipts are NOT legal invoices (see
 * `project_invoicing_model.md`). The shape here mirrors
 * `receiptResponse` in `backend/internal/handler/receipt_handler.go`.
 *
 * The OpenAPI golden marks these endpoints as `additionalProperties:
 * true` (untyped object), so we carry a hand-typed contract here. The
 * `Get<...>` helper is still applied at the call site so the path
 * itself is validated against the OpenAPI document.
 */

/** Snapshot of one party (client / provider / referrer) on a receipt. */
export type ReceiptParty = {
  organization_id: string
  name: string
  /** Always serialized — empty string when not provided by the org. */
  siret: string
  vat: string
  address_line1: string
  address_line2: string
  city: string
  postal_code: string
  country: string
}

/** A single transaction receipt scoped to the caller's org. */
export type Receipt = {
  id: string
  payment_record_id: string
  /** Optional links to the originating proposal/milestone. */
  proposal_id?: string
  milestone_id?: string
  amount_cents: number
  currency: string
  /** ISO-8601 UTC timestamp. */
  created_at: string
  /** Snapshot of the paying client at the time of payment. */
  client: ReceiptParty | null
  /** Snapshot of the receiving provider at the time of payment. */
  provider: ReceiptParty | null
  /** Snapshot of the referrer (apporteur d'affaires), if any. */
  referrer: ReceiptParty | null
  referrer_commission_amount_cents: number
  /**
   * `false` for legacy receipts emitted before snapshotting was
   * deployed — none of the parties / amounts can be trusted in that
   * case and the UI must show a "données indisponibles" badge.
   */
  snapshot_available: boolean
}

/** Cursor-paginated list response. */
export type ReceiptsPage = {
  data: Receipt[]
  next_cursor?: string
}

/** Supported PDF locale. The backend defaults to `fr` when omitted. */
export type ReceiptPdfLanguage = "fr" | "en"
