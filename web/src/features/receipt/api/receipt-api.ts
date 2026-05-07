import { API_BASE_URL, apiClient } from "@/shared/lib/api-client"
import type { Get } from "@/shared/lib/api-paths"
import type { Receipt, ReceiptPdfLanguage, ReceiptsPage } from "../types"

/**
 * Pure async wrappers around the receipt endpoints. Each function
 * issues a single HTTP call. Callers (the TanStack Query hooks in
 * `../hooks/`) own caching, retry and error surfacing — the API
 * layer is transport-only.
 *
 * The backend resolves the caller's organization from the auth
 * cookie, so none of these signatures take an `organization_id`. The
 * path generic on `apiClient<Get<"...">>` validates the path string
 * against the OpenAPI contract; the response cast to the hand-typed
 * `Receipt`/`ReceiptsPage` is needed because the OpenAPI golden
 * marks the body as `additionalProperties: true`.
 */

/** GET /api/v1/receipts?cursor=&limit= — cursor-paginated list. */
export function listReceipts(cursor?: string): Promise<ReceiptsPage> {
  const qs = cursor ? `?cursor=${encodeURIComponent(cursor)}` : ""
  return apiClient<Get<"/api/v1/receipts"> & ReceiptsPage>(
    `/api/v1/receipts${qs}`,
  )
}

/** GET /api/v1/receipts/{id} — single receipt detail. */
export function getReceipt(id: string): Promise<Receipt> {
  return apiClient<Get<"/api/v1/receipts/{id}"> & Receipt>(
    `/api/v1/receipts/${encodeURIComponent(id)}`,
  )
}

/**
 * Returns the absolute URL pointing at the receipt PDF. The endpoint
 * streams `application/pdf` directly with `Content-Disposition:
 * inline`, which lets the browser open the file in a new tab when
 * `target="_blank"` is used.
 *
 * Returns the absolute URL when `NEXT_PUBLIC_API_URL` is set
 * (development), or a relative path when it is empty (production
 * proxy), so the result is always something `<a href>` can consume.
 */
export function getReceiptPdfUrl(
  id: string,
  language: ReceiptPdfLanguage = "fr",
): string {
  const safeId = encodeURIComponent(id)
  const lang = encodeURIComponent(language)
  return `${API_BASE_URL}/api/v1/receipts/${safeId}/pdf?lang=${lang}`
}
