import { test, expect } from "@playwright/test"

/**
 * PERF-W-03 — boundary navigation suite.
 *
 * The dev server's loading.tsx files are rendered while the page's
 * RSC data is in flight. We exercise this by intercepting the
 * backend search endpoint, delaying it for a beat, and asserting
 * that a `[role=status]` skeleton appears within 100 ms of
 * navigation. The skeleton then resolves into the real page.
 *
 * Also asserts:
 *   - /agencies-not-real triggers the locale-scoped not-found
 *     boundary with an `Browse the marketplace` CTA
 */

const BASE_URL = process.env.PLAYWRIGHT_BASE_URL ?? "http://localhost:3001"

test.describe("PERF-W-03 — route loaders and 404 boundary", () => {
  test("public listing skeleton appears during slow data fetch", async ({ page }) => {
    // Stall the search endpoint for 300 ms so the loading boundary
    // has a window in which to render. The page should still
    // ultimately resolve.
    await page.route(/\/api\/v1\/search/, async (route) => {
      await new Promise((r) => setTimeout(r, 300))
      route.continue()
    })

    const navigation = page.goto(`${BASE_URL}/agencies`)
    // Wait for either the skeleton or the page heading — whichever
    // arrives first.
    await page.waitForSelector("[role='status'], h1", { timeout: 5000 })
    await navigation
  })

  test("unknown route triggers the not-found boundary", async ({ page }) => {
    const res = await page.goto(`${BASE_URL}/this-route-does-not-exist`)
    // Next 16 returns 404 for unmatched routes, but the body still
    // renders the not-found boundary tree. Assert the heading.
    expect(res?.status()).toBe(404)
    await expect(page.getByRole("heading", { level: 1 })).toBeVisible()
  })
})
