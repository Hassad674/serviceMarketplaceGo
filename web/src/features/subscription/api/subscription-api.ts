import { apiClient, ApiError } from "@/shared/lib/api-client"
import type {
  BillingCycle,
  CyclePreview,
  SubscribeInput,
  SubscribeResponse,
  Subscription,
  SubscriptionStats,
} from "../types"

/**
 * Pure async functions wrapping the Premium subscription endpoints.
 *
 * Each function issues a single HTTP call. Callers (the TanStack
 * Query hooks in `../hooks/`) decide how to cache, retry, or
 * surface errors — the API layer stays transport-only.
 *
 * The backend resolves the caller identity from the auth cookie
 * (JWT), so none of these signatures accept a `userID` parameter
 * — the server always uses the authenticated user in context.
 */

/** POST /api/v1/subscriptions — start a new Premium checkout. */
export function subscribe(input: SubscribeInput): Promise<SubscribeResponse> {
  return apiClient<SubscribeResponse>("/api/v1/subscriptions", {
    method: "POST",
    body: input,
  })
}

/**
 * GET /api/v1/subscriptions/me — current subscription, or `null`
 * when the caller is on the free tier. The backend returns 404 in
 * that case, which we translate to `null` so the UI can render
 * the free/premium fork without tripping the generic error path.
 */
export async function getMySubscription(): Promise<Subscription | null> {
  try {
    return await apiClient<Subscription>("/api/v1/subscriptions/me")
  } catch (err) {
    if (err instanceof ApiError && err.status === 404) return null
    throw err
  }
}

/** PATCH /api/v1/subscriptions/me/auto-renew — flip cancel_at_period_end. */
export function toggleAutoRenew(autoRenew: boolean): Promise<Subscription> {
  return apiClient<Subscription>("/api/v1/subscriptions/me/auto-renew", {
    method: "PATCH",
    body: { auto_renew: autoRenew },
  })
}

/**
 * PATCH /api/v1/subscriptions/me/billing-cycle — switch monthly↔annual.
 * Both directions are supported. Stripe applies an immediate proration.
 */
export function changeCycle(billingCycle: BillingCycle): Promise<Subscription> {
  return apiClient<Subscription>("/api/v1/subscriptions/me/billing-cycle", {
    method: "PATCH",
    body: { billing_cycle: billingCycle },
  })
}

/**
 * GET /api/v1/subscriptions/me/stats — savings summary since the
 * subscription started. Returns `null` when the caller has no
 * active subscription (same 404-to-null pattern as `getMySubscription`).
 */
export async function getStats(): Promise<SubscriptionStats | null> {
  try {
    return await apiClient<SubscriptionStats>("/api/v1/subscriptions/me/stats")
  } catch (err) {
    if (err instanceof ApiError && err.status === 404) return null
    throw err
  }
}

/**
 * GET /api/v1/subscriptions/me/cycle-preview — computes what Stripe
 * would bill today if the user switched to `billingCycle`, without
 * mutating any state. Used by the manage modal to surface the exact
 * amount BEFORE asking the user to confirm.
 *
 * Upgrade (monthly → annual): amount_due_cents > 0, charged today.
 * Downgrade (annual → monthly): amount_due_cents = 0, scheduled.
 */
export function getCyclePreview(billingCycle: BillingCycle): Promise<CyclePreview> {
  const qs = new URLSearchParams({ billing_cycle: billingCycle }).toString()
  return apiClient<CyclePreview>(`/api/v1/subscriptions/me/cycle-preview?${qs}`)
}

/**
 * GET /api/v1/subscriptions/portal — returns a short-lived URL
 * to the Stripe Customer Portal where the user manages payment
 * methods and views invoices. The caller opens the URL in a new
 * tab — we never embed Stripe's portal in an iframe.
 */
export async function getPortalURL(): Promise<string> {
  const payload = await apiClient<{ url: string }>("/api/v1/subscriptions/portal")
  return payload.url
}
