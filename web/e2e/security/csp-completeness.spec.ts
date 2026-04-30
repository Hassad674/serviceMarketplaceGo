import { expect, test } from "@playwright/test"

/**
 * SEC-03 follow-up — verify the production-style Content-Security-Policy
 * does not block legitimate cross-origin resources we actually use.
 *
 * The CSP allowlist must accommodate (at minimum):
 *   - script-src: 'self', Stripe (js.stripe.com, m.stripe.com)
 *   - connect-src: 'self', backend (NEXT_PUBLIC_API_URL), Stripe API,
 *     Typesense (nodes), LiveKit (wss://), R2/MinIO storage
 *   - img-src: 'self', data:, R2 / MinIO / Cloudinary public buckets
 *   - frame-src: 'self', Stripe (Embedded checkout iframe)
 *
 * If any of these is missing, the legitimate flows break in prod.
 *
 * The test visits a handful of pages and captures any CSP violation
 * the browser surfaces via the Content-Security-Policy-Report-Only
 * header or the `securitypolicyviolation` event. Any violation fails
 * the test with a precise, actionable message.
 */

const PAGES_TO_VISIT = [
  { path: "/", label: "home" },
  { path: "/agencies", label: "public agencies listing" },
  { path: "/freelancers", label: "public freelancers listing" },
  { path: "/login", label: "login" },
  { path: "/register", label: "register" },
]

test.describe("SEC-03 — CSP must not block legitimate resources", () => {
  for (const { path, label } of PAGES_TO_VISIT) {
    test(`no CSP violation on ${label} (${path})`, async ({ page }) => {
      const violations: string[] = []

      // Browser-side capture: the securitypolicyviolation event fires
      // for every blocked resource. We mirror it into a window-level
      // array so the test can read it from Node.
      await page.addInitScript(() => {
        ;(window as unknown as { __cspViolations: string[] }).__cspViolations = []
        document.addEventListener("securitypolicyviolation", (e: SecurityPolicyViolationEvent) => {
          ;(window as unknown as { __cspViolations: string[] }).__cspViolations.push(
            `[${e.violatedDirective}] blocked ${e.blockedURI || "(inline)"}` +
              (e.sourceFile ? ` (source: ${e.sourceFile}:${e.lineNumber})` : ""),
          )
        })
      })

      // Console-side capture as a backup — some browsers route CSP
      // violations through console.error before firing the event.
      page.on("console", (msg) => {
        const text = msg.text()
        if (text.includes("Content Security Policy") || text.includes("Refused to load")) {
          violations.push(text)
        }
      })

      const response = await page.goto(path, { waitUntil: "domcontentloaded" })
      // Give the runtime a tick to dispatch any deferred CSP errors
      // that happen during late-stage hydration.
      await page.waitForTimeout(800)

      const browserViolations = (await page.evaluate(
        () => (window as unknown as { __cspViolations: string[] }).__cspViolations || [],
      )) as string[]

      const all = [...violations, ...browserViolations]

      if (all.length > 0) {
        // Make the failure message obvious + actionable.
        throw new Error(
          `${all.length} CSP violation(s) on ${path}:\n  - ${all.join("\n  - ")}\n` +
            `Fix: extend the CSP in backend/internal/handler/middleware/security_headers.go ` +
            `and web/next.config.ts to allowlist the blocked source(s).`,
        )
      }

      // Sanity: the page actually rendered.
      expect(response?.status()).toBeLessThan(500)
    })
  }
})
