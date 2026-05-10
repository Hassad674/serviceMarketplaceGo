import { test, expect } from "@playwright/test"
import {
  registerEnterprise,
  registerProvider,
} from "./helpers/auth"

// stats-page.spec.ts — guards the /stats route added by
// R-DASH-2026-05-10. Enterprise must never reach the page (redirect
// to /dashboard); Provider sees the period selector + chart panels +
// keywords table even with empty data (graceful empty states).

test.describe("/stats page", () => {
  test("Enterprise visiting /stats redirects to /dashboard", async ({ page }) => {
    await registerEnterprise(page)
    await page.goto("/en/stats")
    await page.waitForURL(/\/dashboard/, { timeout: 10000 })
    await expect(page).toHaveURL(/\/dashboard/)
  })

  test("Provider visiting /stats sees period selector + charts + keywords", async ({ page }) => {
    await registerProvider(page)
    await page.goto("/en/stats")
    await expect(page.getByRole("heading", { level: 1 })).toBeVisible({
      timeout: 15000,
    })
    // Period selector
    await expect(page.getByRole("button", { name: /^7 days?$/i })).toBeVisible()
    await expect(page.getByRole("button", { name: /^30 days?$/i })).toBeVisible()
    await expect(page.getByRole("button", { name: /^90 days?$/i })).toBeVisible()
    // Charts (each rendered as a role="img")
    await expect(page.getByRole("img", { name: "Profile views" })).toBeVisible()
    await expect(page.getByRole("img", { name: "Search impressions" })).toBeVisible()
    // Top keywords section is rendered (heading)
    await expect(page.getByRole("heading", { name: /Top keywords/i })).toBeVisible()
  })

  test("changing the period updates the URL ?period= param", async ({ page }) => {
    await registerProvider(page)
    await page.goto("/en/stats")
    const ninetyDays = page.getByRole("button", { name: /^90 days?$/i })
    await ninetyDays.click()
    await expect(page).toHaveURL(/period=90/)
  })
})
