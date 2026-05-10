import { test, expect } from "@playwright/test"
import {
  registerProvider,
  registerAgency,
  registerEnterprise,
} from "./helpers/auth"

// dashboard-role-aware.spec.ts — locks in the role-aware dashboard
// regression contract introduced by R-DASH-2026-05-10:
//
//   - Provider / Agency see the visibility stat strip + a link to /stats.
//   - Enterprise sees the recruitments stat strip and NEVER the
//     visibility cards.
//   - Clicking "View detailed statistics" on a Provider/Agency
//     dashboard navigates to /stats.
//
// All assertions are EN-locale because the helpers force /en at
// registration time.

test.describe("Dashboard role-aware layout", () => {
  test("Provider sees visibility cards (not Enterprise ones)", async ({ page }) => {
    await registerProvider(page)
    await expect(page.getByText("Profile views")).toBeVisible({ timeout: 10000 })
    await expect(page.getByText("Search impressions")).toBeVisible()
    await expect(page.getByText("Average search position")).toBeVisible()
    // Enterprise-only labels must never leak
    await expect(page.getByText("Active recruitments")).not.toBeVisible()
    await expect(page.getByText("Applications received")).not.toBeVisible()
  })

  test("Agency sees visibility cards", async ({ page }) => {
    await registerAgency(page)
    await expect(page.getByText("Profile views")).toBeVisible({ timeout: 10000 })
    await expect(page.getByText("Search impressions")).toBeVisible()
    await expect(page.getByText("Active recruitments")).not.toBeVisible()
  })

  test("Enterprise sees recruitment cards (not visibility ones)", async ({ page }) => {
    await registerEnterprise(page)
    await expect(page.getByText("Active recruitments")).toBeVisible({ timeout: 10000 })
    await expect(page.getByText("Applications received")).toBeVisible()
    await expect(page.getByText("To review")).toBeVisible()
    // Visibility labels must never leak
    await expect(page.getByText("Profile views")).not.toBeVisible()
    await expect(page.getByText("Search impressions")).not.toBeVisible()
  })

  test("Provider link to /stats navigates to the deep dive page", async ({ page }) => {
    await registerProvider(page)
    const link = page.getByRole("link", { name: /detailed statistics/i })
    await expect(link).toBeVisible({ timeout: 10000 })
    await link.click()
    await expect(page).toHaveURL(/\/stats/)
  })
})
