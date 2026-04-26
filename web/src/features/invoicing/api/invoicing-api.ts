import { API_BASE_URL, apiClient } from "@/shared/lib/api-client"
import type {
  BillingProfileSnapshot,
  CurrentMonthAggregate,
  InvoicesPage,
  UpdateBillingProfileInput,
  VIESResult,
} from "../types"

/**
 * Pure async wrappers around the invoicing endpoints. Each function
 * issues a single HTTP call. Callers (the TanStack Query hooks in
 * `../hooks/`) own caching, retry, and error surfacing — the API
 * layer is transport-only.
 *
 * The backend resolves the caller's organization from the auth
 * cookie, so none of these signatures take an `organization_id`.
 */

/** GET /api/v1/me/billing-profile */
export function fetchBillingProfile(): Promise<BillingProfileSnapshot> {
  return apiClient<BillingProfileSnapshot>("/api/v1/me/billing-profile")
}

/** PUT /api/v1/me/billing-profile — partial saves are accepted server-side. */
export function updateBillingProfile(
  input: UpdateBillingProfileInput,
): Promise<BillingProfileSnapshot> {
  return apiClient<BillingProfileSnapshot>("/api/v1/me/billing-profile", {
    method: "PUT",
    body: input,
  })
}

/** POST /api/v1/me/billing-profile/sync-from-stripe */
export function syncBillingProfileFromStripe(): Promise<BillingProfileSnapshot> {
  return apiClient<BillingProfileSnapshot>(
    "/api/v1/me/billing-profile/sync-from-stripe",
    { method: "POST" },
  )
}

/** POST /api/v1/me/billing-profile/validate-vat — VIES round-trip. */
export function validateBillingProfileVAT(): Promise<VIESResult> {
  return apiClient<VIESResult>("/api/v1/me/billing-profile/validate-vat", {
    method: "POST",
  })
}

/** GET /api/v1/me/invoices?cursor= — cursor-paginated list. */
export function fetchInvoices(cursor?: string): Promise<InvoicesPage> {
  const qs = cursor ? `?cursor=${encodeURIComponent(cursor)}` : ""
  return apiClient<InvoicesPage>(`/api/v1/me/invoices${qs}`)
}

/**
 * GET /api/v1/me/invoices/:id/pdf URL.
 *
 * The handler returns a 302 to a short-lived presigned R2 URL. We
 * never fetch this in JS — the browser follows the redirect natively
 * when the user clicks the link, which avoids dragging the binary
 * through our process and lets the download dialog fire normally.
 *
 * Returns the absolute URL when `NEXT_PUBLIC_API_URL` is set
 * (development), or a relative path when it is empty (production
 * proxy), so the result is always something `<a href>` can consume.
 */
export function getInvoicePDFURL(id: string): string {
  return `${API_BASE_URL}/api/v1/me/invoices/${id}/pdf`
}

/** GET /api/v1/me/invoicing/current-month — running fee total. */
export function fetchCurrentMonthAggregate(): Promise<CurrentMonthAggregate> {
  return apiClient<CurrentMonthAggregate>("/api/v1/me/invoicing/current-month")
}
