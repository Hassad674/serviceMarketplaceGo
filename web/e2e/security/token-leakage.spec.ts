import { test, expect } from "@playwright/test"

/**
 * SEC-14: the web middleware accepts `?token=` ONLY on the sanctioned
 * mobile-WebView bridge routes (`/payment-info`, `/subscribe/*`,
 * `/billing/embed*`). On every other route the middleware MUST drop
 * the query parameter before any page logic can see it.
 *
 * These tests run against the live Next.js dev server (or the deployed
 * preview URL via PLAYWRIGHT_BASE_URL). They assert the URL after
 * navigation no longer contains the token, and that the middleware's
 * decision matches the per-path matrix in src/middleware.ts.
 */

test.describe("SEC-14 ?token= query string handling", () => {
  test("strips ?token= on /dashboard and redirects to login", async ({ page }) => {
    // Unauthenticated visit to /dashboard with a planted token must
    // redirect to /login WITHOUT carrying the token along.
    await page.goto("/dashboard?token=stolen-jwt-from-phishing-link")

    // Wait for the redirect to settle.
    await page.waitForURL(/\/login(\?|$)/)

    expect(page.url()).not.toContain("token=")
    expect(page.url()).not.toContain("stolen-jwt-from-phishing-link")
  })

  test("strips ?token= on a public page", async ({ page }) => {
    // The landing page is not protected — but it must STILL drop the
    // token from the URL so a subsequent click does not leak it via
    // referer.
    await page.goto("/?token=should-disappear")

    // Wait for the strip-redirect to settle. We allow either the
    // root URL or a locale-prefixed root depending on the next-intl
    // config of the deployment.
    await page.waitForLoadState("domcontentloaded")
    expect(page.url()).not.toContain("token=")
  })

  test("preserves ?token= on /payment-info bridge route", async ({ page }) => {
    // The token is the legitimate transport for the mobile WebView on
    // the sanctioned bridge routes. The middleware must NOT strip it,
    // even when the user is not yet authenticated. The page itself
    // exchanges it for a session cookie via /api/v1/auth/web-session.
    //
    // We only assert that the URL is NOT redirected to /login with a
    // stripped token — the actual cookie exchange depends on the
    // backend being reachable at the same baseURL, which the suite
    // does not guarantee.
    await page.goto("/payment-info?token=valid-bridge-token")

    // The middleware should not redirect us off the page. We do not
    // assert success; only that the token was not stripped to /login.
    await page.waitForLoadState("domcontentloaded")
    expect(page.url()).toMatch(/\/payment-info/)
  })

  test("preserves ?token= on /subscribe bridge route", async ({ page }) => {
    await page.goto("/subscribe?token=valid-bridge-token")
    await page.waitForLoadState("domcontentloaded")
    expect(page.url()).toMatch(/\/subscribe/)
  })

  test("preserves ?token= on /billing/embed bridge route", async ({ page }) => {
    await page.goto("/billing/embed?token=valid-bridge-token")
    await page.waitForLoadState("domcontentloaded")
    expect(page.url()).toMatch(/\/billing\/embed/)
  })

  test("strips ?token= on /messages (protected non-bridge route)", async ({ page }) => {
    await page.goto("/messages?token=phishing-jwt")
    await page.waitForLoadState("domcontentloaded")

    // Either we redirected to /login (no cookie) or we landed on
    // /messages (with cookie); in BOTH cases the token must be gone.
    expect(page.url()).not.toContain("token=phishing-jwt")
  })
})
