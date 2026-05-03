import { apiClient } from "@/shared/lib/api-client"
import type { Get, Post, Put } from "@/shared/lib/api-paths"
import type {
  BillingProfileSnapshot,
  CurrentMonthAggregate,
  UpdateBillingProfileInput,
  VIESResult,
} from "@/shared/types/billing-profile"

/**
 * Shared billing-profile API (P9 — `BillingProfileCompletionModal` and
 * `CurrentMonthAggregate` are consumed cross-feature by wallet, so the
 * underlying API calls live in `shared/`). Lifted from
 * `features/invoicing/api/invoicing-api`.
 *
 * The backend resolves the caller's organization from the auth cookie,
 * so none of these signatures take an `organization_id`.
 */

/** GET /api/v1/me/billing-profile */
export function fetchBillingProfile(): Promise<BillingProfileSnapshot> {
  return apiClient<Get<"/api/v1/me/billing-profile"> & BillingProfileSnapshot>("/api/v1/me/billing-profile")
}

/** PUT /api/v1/me/billing-profile — partial saves are accepted server-side. */
export function updateBillingProfile(
  input: UpdateBillingProfileInput,
): Promise<BillingProfileSnapshot> {
  return apiClient<Put<"/api/v1/me/billing-profile"> & BillingProfileSnapshot>("/api/v1/me/billing-profile", {
    method: "PUT",
    body: input,
  })
}

/** POST /api/v1/me/billing-profile/sync-from-stripe */
export function syncBillingProfileFromStripe(): Promise<BillingProfileSnapshot> {
  return apiClient<Post<"/api/v1/me/billing-profile/sync-from-stripe"> & BillingProfileSnapshot>(
    "/api/v1/me/billing-profile/sync-from-stripe",
    { method: "POST" },
  )
}

/** POST /api/v1/me/billing-profile/validate-vat — VIES round-trip. */
export function validateBillingProfileVAT(): Promise<VIESResult> {
  return apiClient<Post<"/api/v1/me/billing-profile/validate-vat"> & VIESResult>("/api/v1/me/billing-profile/validate-vat", {
    method: "POST",
  })
}

/** GET /api/v1/me/invoicing/current-month — running fee total. */
export function fetchCurrentMonthAggregate(): Promise<CurrentMonthAggregate> {
  return apiClient<Get<"/api/v1/me/invoicing/current-month"> & CurrentMonthAggregate>("/api/v1/me/invoicing/current-month")
}
