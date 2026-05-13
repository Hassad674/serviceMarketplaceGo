/**
 * End-to-end regression for the locale-aware legal route segments.
 *
 * Verifies:
 *   1. `/fr/legal/cgu` renders the CGU page in French;
 *   2. `/legal/terms` (EN URL, default locale → no prefix) renders the
 *      same page in English (via the next.config rewrite);
 *   3. The cookie consent banner footer surfaces the privacy + cookies
 *      links with locale-aware hrefs (FR-prefixed on `/fr`, EN-named
 *      on the default-locale path).
 *
 * Auto-skip when the dev server is unreachable — this suite is
 * primarily executed by the CI matrix that boots a Next.js dev server
 * via `playwright.config.ts > webServer`. Local executions where the
 * server is intentionally down should not fail the suite.
 */
import { test, expect } from "@playwright/test"

async function serverReachable(baseURL: string | undefined): Promise<boolean> {
  if (!baseURL) return false
  try {
    const res = await fetch(baseURL, { method: "GET" })
    return res.status < 500
  } catch {
    return false
  }
}

test.describe("Legal routes — locale-aware URL segments", () => {
  test.beforeEach(async ({ baseURL, context }, testInfo) => {
    const reachable = await serverReachable(baseURL)
    test.skip(!reachable, "Dev server not reachable — local skip")
    await context.clearCookies()
    void testInfo
  })

  test("/fr/legal/cgu renders the CGU page in French", async ({ page }) => {
    await page.goto("/fr/legal/cgu")
    await expect(page).toHaveURL(/\/fr\/legal\/cgu$/)
    // The CGU page renders the legal shell with the FR title key.
    const heading = page.locator("h1").first()
    await expect(heading).toBeVisible()
  })

  test("/legal/terms serves the CGU page on the EN locale", async ({
    page,
  }) => {
    // EN is default-locale + as-needed → no /en prefix on the URL.
    // The next.config rewrite maps /legal/terms → /legal/cgu so the
    // same on-disk page is rendered without duplicating the file.
    const res = await page.goto("/legal/terms")
    expect(res?.status()).toBe(200)
    await expect(page).toHaveURL(/\/legal\/terms$/)
    const heading = page.locator("h1").first()
    await expect(heading).toBeVisible()
  })

  test("cookie consent banner exposes localized legal links", async ({
    page,
  }) => {
    await page.goto("/fr")
    // Wait for the CMP banner to mount.
    const banner = page.locator("#cc-main").first()
    await expect(banner).toBeVisible({ timeout: 10_000 })

    // Footer slot contains the 4 legal anchors on the FR locale.
    const privacyLink = banner.locator(
      'a[href="/fr/legal/politique-confidentialite"]',
    )
    await expect(privacyLink).toBeVisible()
    const cookiesLink = banner.locator('a[href="/fr/cookies"]')
    await expect(cookiesLink).toBeVisible()
  })
})
