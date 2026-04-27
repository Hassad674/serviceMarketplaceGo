/**
 * Stripe.js loader, memoised at module level so `loadStripe` runs at
 * most once per page-load. Returns a Promise<Stripe | null> the
 * @stripe/react-stripe-js providers expect.
 *
 * The publishable key is read from `NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY`
 * — the value is safe to ship to the browser (it's the public
 * counterpart to the server-side secret key). When the env var is
 * absent we still return a non-null promise — but `loadStripe` will
 * reject internally and any `<EmbeddedCheckoutProvider>` mounted
 * with that promise will surface the error in its own boundary.
 *
 * Consumers should import `stripePromise` directly:
 *
 *   import { stripePromise } from "@/shared/lib/stripe-client"
 *   <EmbeddedCheckoutProvider stripe={stripePromise} options={...}>
 */

import { loadStripe, type Stripe } from "@stripe/stripe-js"

const publishableKey = process.env.NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY ?? ""

export const stripePromise: Promise<Stripe | null> = loadStripe(publishableKey)
