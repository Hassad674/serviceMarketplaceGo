import { adminApi } from "@/shared/lib/api-client"
import type { AdminInvoiceFilters, AdminInvoiceListResponse } from "../types"

// fetchAdminInvoices serializes the filter struct into the backend's
// query-string contract. Empty fields are omitted entirely so the
// backend treats them as "no filter".
export function fetchAdminInvoices(
  filters: AdminInvoiceFilters,
): Promise<AdminInvoiceListResponse> {
  const params = new URLSearchParams()
  if (filters.recipient_org_id) params.set("recipient_org_id", filters.recipient_org_id)
  if (filters.status) params.set("status", filters.status)
  if (filters.date_from) params.set("date_from", filters.date_from)
  if (filters.date_to) params.set("date_to", filters.date_to)
  if (filters.min_amount_cents) params.set("min_amount_cents", filters.min_amount_cents)
  if (filters.max_amount_cents) params.set("max_amount_cents", filters.max_amount_cents)
  if (filters.search) params.set("search", filters.search)
  if (filters.cursor) params.set("cursor", filters.cursor)
  params.set("limit", "20")
  const qs = params.toString()
  return adminApi<AdminInvoiceListResponse>(
    `/api/v1/admin/invoices${qs ? `?${qs}` : ""}`,
  )
}

// getInvoicePDFRedirect returns the absolute URL the operator should
// open to get the presigned PDF. The backend responds with a 302 to a
// 5-minute presigned R2 URL — opening the URL in a new tab does the
// redirect transparently. We add the bearer token via a query param
// would expose the token; instead we assume the operator's session
// cookie or the localStorage bearer token is read at request time. To
// keep the redirect flow simple and avoid leaking the bearer token, we
// fetch the PDF endpoint with the token, follow the redirect, and open
// the eventual presigned URL in a new tab.
export async function openInvoicePDF(id: string, isCreditNote: boolean): Promise<string> {
  // We cannot rely on adminApi here because it parses JSON and we want
  // the redirect target. Re-implement the auth header read once + do a
  // fetch with redirect: "follow".
  const apiUrl = (import.meta.env.VITE_API_URL as string | undefined) ?? "http://localhost:8083"
  const token = localStorage.getItem("admin_token")
  const typeParam = isCreditNote ? "credit_note" : "invoice"
  const url = `${apiUrl}/api/v1/admin/invoices/${id}/pdf?type=${typeParam}`
  const res = await fetch(url, {
    method: "GET",
    headers: token ? { Authorization: `Bearer ${token}` } : {},
    redirect: "follow",
  })
  if (!res.ok) {
    throw new Error(`Failed to open PDF (${res.status})`)
  }
  // After following the redirect, res.url is the presigned R2 URL.
  return res.url
}
