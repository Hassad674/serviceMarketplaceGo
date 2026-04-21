// Types mirroring the backend Premium subscription DTOs
// (internal/handler/subscription_handler.go). The Subscription
// status mirrors Stripe's own set: incomplete, active, past_due,
// canceled, unpaid. cancel_at_period_end = true means the user
// has turned auto-renew OFF — the plan stays active until the
// end of the paid period, then expires naturally.

export type Plan = "freelance" | "agency"

export type BillingCycle = "monthly" | "annual"

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

export type SubscribeResponse = {
  checkout_url: string
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
}

export type SubscriptionStats = {
  saved_fee_cents: number
  saved_count: number
  /** ISO-8601 UTC */
  since: string
}
