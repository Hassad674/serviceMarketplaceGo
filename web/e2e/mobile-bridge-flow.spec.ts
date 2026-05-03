/**
 * mobile-bridge-flow.spec.ts
 *
 * E2E coverage for the mobile-WebView bridge: the routes that the Flutter
 * app loads inside an in-app WebView and the `?token=` handshake that
 * carries the mobile JWT into a session cookie on the web side.
 *
 * Critical contracts being locked down:
 *   - /subscribe/embed accepts a mobile token in the query string and
 *     boots the embedded checkout without an additional sign-in
 *   - /subscribe/return is the documented landing page after Stripe
 *     completes; the WebView intercepts it to dismiss itself
 *   - /account/confirm-deletion and /account/cancel-deletion render
 *     publicly so the mobile email-link flow works
 *
 * Gated by PLAYWRIGHT_E2E=1 so unit-only CI runs do not pay the
 * round-trip cost.
 */
import { test, expect } from "@playwright/test"

test.skip(
  process.env.PLAYWRIGHT_E2E !== "1",
  "Gated behind PLAYWRIGHT_E2E=1 (requires running backend)",
)

test.describe("Mobile bridge — public routes", () => {
  test("/subscribe/return renders without auth (Flutter intercepts)", async ({ page }) => {
    // Mobile WebViews navigate through /subscribe/return to detect that
    // Stripe finished. The page does not need to do anything visible —
    // it just has to NOT redirect to /login (which would break the
    // WebView dismiss heuristic in the Flutter side).
    const response = await page.goto("/subscribe/return?session_id=cs_test_123")
    expect(response?.status()).toBeLessThan(400)
    // No redirect to /login.
    expect(page.url()).not.toContain("/login")
  })

  test("/account/confirm-deletion renders publicly", async ({ page }) => {
    const response = await page.goto("/account/confirm-deletion?token=fake-jwt")
    expect(response?.status()).toBeLessThan(400)
    expect(page.url()).not.toContain("/login")
  })

  test("/account/cancel-deletion renders publicly", async ({ page }) => {
    const response = await page.goto("/account/cancel-deletion?token=fake-jwt")
    expect(response?.status()).toBeLessThan(400)
    expect(page.url()).not.toContain("/login")
  })
})

test.describe("Mobile bridge — token handshake", () => {
  test("/subscribe/embed?token=… does not error out", async ({ page }) => {
    // The page tries to mount the Stripe checkout. With a fake token
    // the backend will refuse, but the page itself must render —
    // refusing the WebView would surface as a blank screen on mobile.
    const response = await page.goto("/subscribe/embed?token=fake-jwt&plan=premium&billing_cycle=monthly")
    expect(response?.status()).toBeLessThan(500)
  })

  test("invalid token does not surface a stack trace", async ({ page }) => {
    await page.goto("/subscribe/embed?token=bogus")
    const html = await page.content()
    // Stack traces never reach the user — we redact via the error
    // boundary. The server returns the empty / loading state.
    expect(html).not.toMatch(/at .+\(.+:\d+:\d+\)/)
  })
})

test.describe("Mobile bridge — return URL safety", () => {
  test("redirect on /subscribe/return is internal-only (no open redirect)", async ({ page }) => {
    // Open redirect protection: passing return_to=external://… must
    // not let the page navigate off-domain.
    await page.goto("/subscribe/return?return_to=https://evil.example.com")
    // After loading, the page may redirect to dashboard or stay put —
    // it MUST NOT navigate to evil.example.com.
    expect(page.url()).not.toContain("evil.example.com")
  })
})
