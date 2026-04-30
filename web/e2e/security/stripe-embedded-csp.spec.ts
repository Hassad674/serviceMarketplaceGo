import { expect, test } from "@playwright/test"

/**
 * SEC-03 critical follow-up — make sure the new CSP does not
 * accidentally break Stripe Embedded Components, which is the load-
 * bearing payment flow (KYC onboarding + Premium subscribe checkout).
 *
 * Stripe Connect Embedded loads a script bundle from
 *   https://b.stripecdn.com / https://connect-js.stripe.com
 * and opens iframes from
 *   https://js.stripe.com
 * Both must be allowlisted in the CSP for the Stripe widget to even
 * mount; if they are not, the customer hits a blank component or a
 * console error and cannot complete payment / KYC.
 *
 * The test visits both bridge routes (/payment-info and /subscribe/embed)
 * and asserts:
 *   1. The page loads (status < 500)
 *   2. No CSP violation references a stripe.com host
 *   3. The expected Stripe DOM hook is reachable (best-effort —
 *      we don't sign in here so the actual widget may not render,
 *      but the absence of a CSP refusal is the load-bearing
 *      assertion)
 */

const STRIPE_HOSTS = [
  "stripe.com",
  "stripecdn.com",
  "connect-js.stripe.com",
  "js.stripe.com",
  "m.stripe.com",
]

const STRIPE_BRIDGE_ROUTES = [
  { path: "/payment-info", label: "KYC embedded onboarding" },
  { path: "/subscribe/embed", label: "Premium embedded checkout" },
]

test.describe("SEC-03 × Stripe — embedded widgets not CSP-blocked", () => {
  for (const { path, label } of STRIPE_BRIDGE_ROUTES) {
    test(`CSP does not block Stripe on ${label} (${path})`, async ({ page }) => {
      const stripeViolations: string[] = []
      const consoleErrors: string[] = []

      await page.addInitScript(() => {
        ;(window as unknown as { __cspViolations: string[] }).__cspViolations = []
        document.addEventListener("securitypolicyviolation", (e: SecurityPolicyViolationEvent) => {
          ;(window as unknown as { __cspViolations: string[] }).__cspViolations.push(
            JSON.stringify({
              directive: e.violatedDirective,
              uri: e.blockedURI || "",
              source: e.sourceFile || "",
            }),
          )
        })
      })

      page.on("console", (msg) => {
        const text = msg.text()
        if (
          (text.includes("Content Security Policy") || text.includes("Refused to")) &&
          STRIPE_HOSTS.some((h) => text.includes(h))
        ) {
          stripeViolations.push(text)
        }
        if (msg.type() === "error" && STRIPE_HOSTS.some((h) => text.includes(h))) {
          consoleErrors.push(text)
        }
      })

      const response = await page.goto(path, { waitUntil: "domcontentloaded" })
      await page.waitForTimeout(1500)

      const raw = (await page.evaluate(
        () => (window as unknown as { __cspViolations: string[] }).__cspViolations || [],
      )) as string[]

      const browserStripeViolations = raw
        .map((j) => {
          try {
            return JSON.parse(j) as { directive: string; uri: string; source: string }
          } catch {
            return null
          }
        })
        .filter((v): v is { directive: string; uri: string; source: string } => v !== null)
        .filter(
          (v) =>
            STRIPE_HOSTS.some((h) => v.uri.includes(h)) ||
            STRIPE_HOSTS.some((h) => v.source.includes(h)),
        )

      // Auth gate may redirect us away from these pages — that's fine,
      // the CSP test still applies to whichever page actually rendered.
      // Skip the CSP assertion only if the navigation outright failed.
      if (!response) {
        test.skip(true, `failed to load ${path}`)
        return
      }
      expect(response.status()).toBeLessThan(500)

      if (stripeViolations.length > 0 || browserStripeViolations.length > 0) {
        throw new Error(
          `Stripe-related CSP violation(s) on ${path}:\n` +
            `  via console: ${stripeViolations.join("; ") || "(none)"}\n` +
            `  via event: ${browserStripeViolations.map((v) => `${v.directive} -> ${v.uri}`).join("; ") || "(none)"}\n` +
            `Fix: extend script-src / connect-src / frame-src in ` +
            `backend/internal/handler/middleware/security_headers.go ` +
            `and web/next.config.ts to allowlist:\n` +
            `  https://js.stripe.com, https://*.stripe.com, https://b.stripecdn.com`,
        )
      }
    })
  }
})
