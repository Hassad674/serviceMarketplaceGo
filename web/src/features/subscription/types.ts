// Types mirroring the backend Premium subscription DTOs
// (internal/handler/subscription_handler.go). The Subscription
// status mirrors Stripe's own set: incomplete, active, past_due,
// canceled, unpaid. cancel_at_period_end = true means the user
// has turned auto-renew OFF — the plan stays active until the
// end of the paid period, then expires naturally.

// `Plan` and `BillingCycle` are shared with the proposal feature (P9
// — `UpgradeCta` / `UpgradeModal` are rendered inline inside proposal
// flows). They live in `@/shared/types/subscription` and are
// re-exported here so existing intra-feature imports keep working.
import type { Plan, BillingCycle } from "@/shared/types/subscription"
export type { Plan, BillingCycle }

export type SubscriptionStatus =
  | "incomplete"
  | "active"
  | "past_due"
  | "canceled"
  | "unpaid"

export type SubscribeInput = {
  plan: Plan
  billing_cycle: BillingCycle
  auto_renew: boolean
}

/**
 * Backend now returns a Stripe Embedded Checkout session secret instead
 * of a hosted URL. The web client mounts it via @stripe/react-stripe-js
 * (`<EmbeddedCheckoutProvider>` + `<EmbeddedCheckout>`); the mobile
 * client opens a WebView pointed at our /subscribe/embed page which
 * does the same thing inside the app.
 */
export type SubscribeResponse = {
  client_secret: string
}

export type Subscription = {
  id: string
  plan: Plan
  billing_cycle: BillingCycle
  status: SubscriptionStatus
  /** ISO-8601 UTC */
  current_period_start: string
  /** ISO-8601 UTC */
  current_period_end: string
  /** TRUE means auto-renew is OFF */
  cancel_at_period_end: boolean
  /** ISO-8601 UTC */
  started_at: string
  grace_period_ends_at?: string
  canceled_at?: string
  /**
   * When set, the user has scheduled a cycle switch that takes effect
   * at `pending_cycle_effective_at`. Until then the current cycle
   * stays in force (e.g. annual keeps running until its end date).
   * Both fields are populated together or both absent.
   */
  pending_billing_cycle?: BillingCycle
  /** ISO-8601 UTC — when the pending cycle takes over. */
  pending_cycle_effective_at?: string
}

export type SubscriptionStats = {
  saved_fee_cents: number
  saved_count: number
  /** ISO-8601 UTC */
  since: string
}

/**
 * Invoice preview returned by GET /subscriptions/me/cycle-preview.
 *
 * amount_due_cents > 0 → the user is charged that amount today
 * (upgrade path). 0 means no immediate charge (downgrade is scheduled).
 * prorate_immediately mirrors the backend flag so the UI can switch
 * copy ("Tu seras facturé …" vs "Aucun débit aujourd'hui, bascule le …").
 */
export type CyclePreview = {
  amount_due_cents: number
  currency: string
  /** ISO-8601 UTC */
  period_start: string
  /** ISO-8601 UTC */
  period_end: string
  prorate_immediately: boolean
}
