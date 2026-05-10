// Centralised conversion-event helpers — the single shim every
// feature calls so analytics SDKs (PostHog + GA4) coexist without
// double-instrument bugs at the call site.
//
// PostHog = product analytics ("what they do") — captured server-side
//   for guaranteed delivery on `auth.user_registered` and
//   `proposal.payment_succeeded`. The browser fires the same events
//   so dashboards keep working when the backend's idempotency drops
//   a duplicate.
// GA4 = acquisition analytics ("where they came from") via Search
//   Console. Fires the GA4-canonical event names (`sign_up`,
//   `purchase`, `generate_lead`, `search`) so the dashboards Google
//   ships out of the box light up automatically.
//
// All helpers are no-ops when the SDK is not initialised (missing
// env, server-side, consent not granted). Call sites can fire them
// unconditionally — this is by design so feature code never branches
// on consent or env state.

import { captureGAEvent } from "@/shared/lib/ga"
import { captureEvent as capturePosthogEvent } from "@/shared/lib/posthog"

/** Auth — successful registration. Mirrors backend `auth.user_registered`. */
export function trackSignUp(properties?: { method?: string; role?: string }) {
  const method = properties?.method ?? "email"
  capturePosthogEvent("auth.register_completed", {
    method,
    role: properties?.role,
  })
  captureGAEvent("sign_up", {
    method,
    role: properties?.role,
  })
}

/** Stripe payment success — GA4 ecommerce schema. */
export interface PurchaseProperties {
  value: number
  currency: string
  transactionId: string
  items?: Array<{
    item_id: string
    item_name: string
    price?: number
    quantity?: number
  }>
}

export function trackPurchase(props: PurchaseProperties) {
  capturePosthogEvent("proposal.payment_succeeded_client", {
    value: props.value,
    currency: props.currency,
    transaction_id: props.transactionId,
  })
  captureGAEvent("purchase", {
    value: props.value,
    currency: props.currency,
    transaction_id: props.transactionId,
    items: props.items ?? [],
  })
}

/** Lead — user clicked "Send message" on a public profile. */
export function trackLead(props: { profileId: string; persona: string }) {
  capturePosthogEvent("public_profile.send_message_clicked", {
    profile_id: props.profileId,
    persona: props.persona,
  })
  captureGAEvent("generate_lead", {
    profile_id: props.profileId,
    persona: props.persona,
  })
}

/** Search submit — landing or /search results page. */
export function trackSearch(props: {
  searchTerm: string
  persona: string
  filtersCount?: number
}) {
  const filtersCount = props.filtersCount ?? 0
  capturePosthogEvent("search.executed", {
    query: props.searchTerm,
    persona: props.persona,
    filters_applied: filtersCount,
  })
  captureGAEvent("search", {
    search_term: props.searchTerm,
    persona: props.persona,
    filters_count: filtersCount,
  })
}
