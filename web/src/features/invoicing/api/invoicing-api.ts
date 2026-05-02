import { API_BASE_URL, apiClient } from "@/shared/lib/api-client"
import type { InvoicesPage } from "../types"

/**
 * Pure async wrappers around the invoicing endpoints. Each function
 * issues a single HTTP call. Callers (the TanStack Query hooks in
 * `../hooks/`) own caching, retry, and error surfacing — the API
 * layer is transport-only.
 *
 * The backend resolves the caller's organization from the auth
 * cookie, so none of these signatures take an `organization_id`.
 */

// Billing-profile endpoints (`fetchBillingProfile`, `updateBillingProfile`,
// `syncBillingProfileFromStripe`, `validateBillingProfileVAT`,
// `fetchCurrentMonthAggregate`) live in
// `@/shared/lib/billing-profile/billing-profile-api` (P9 — wallet
// renders the completion gate and current-month block, so the data
// layer is shared). Re-exported here for back-compat.
export {
  fetchBillingProfile,
  updateBillingProfile,
  syncBillingProfileFromStripe,
  validateBillingProfileVAT,
  fetchCurrentMonthAggregate,
} from "@/shared/lib/billing-profile/billing-profile-api"

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
