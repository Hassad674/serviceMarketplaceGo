import { test, expect } from "@playwright/test"
import { registerEnterprise, clearAuth } from "./helpers/auth"

// ---------------------------------------------------------------------------
// SearchFilterSidebar — Phase 3 god-component split smoke
//
// The 758-line shared/components/search/search-filter-sidebar.tsx
// has been decomposed into a 139-line orchestrator + 6 focused
// section components + a primitives module. These tests prove that
// the user-facing filter UI on /search is unchanged after the split.
// ---------------------------------------------------------------------------

test.describe("Search filter sidebar — Phase 3 split smoke", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/")
    await clearAuth(page)
  })

  test("freelancer search renders the filter sidebar with every section", async ({
    page,
  }) => {
    await registerEnterprise(page)
    await page.goto("/en/search?type=freelancer")
    await page.waitForLoadState("networkidle")

    // The sidebar is an aside with the "Filters" landmark.
    const sidebar = page.getByRole("complementary", { name: "Filters" })
    await expect(sidebar).toBeVisible({ timeout: 15_000 })

    // Each section heading from the new components
    await expect(
      page.getByRole("heading", { name: "Availability" }),
    ).toBeVisible()
    await expect(page.getByRole("heading", { name: "Location" })).toBeVisible()
    await expect(page.getByRole("heading", { name: "Languages" })).toBeVisible()
    await expect(page.getByRole("heading", { name: "Expertise" })).toBeVisible()
    await expect(page.getByRole("heading", { name: "Skills" })).toBeVisible()
    await expect(page.getByRole("heading", { name: "Work mode" })).toBeVisible()
  })

  test("clicking an availability pill toggles aria-pressed", async ({ page }) => {
    await registerEnterprise(page)
    await page.goto("/en/search?type=freelancer")
    await page.waitForLoadState("networkidle")

    // Sidebar takes a moment to render
    await expect(
      page.getByRole("complementary", { name: "Filters" }),
    ).toBeVisible({ timeout: 15_000 })
    const nowPill = page
      .getByRole("complementary", { name: "Filters" })
      .getByRole("button", { name: "Now" })
    await expect(nowPill).toHaveAttribute("aria-pressed", "false")
    await nowPill.click()
    // After click, parent state flips, prop re-renders the pill.
    await expect(nowPill).toHaveAttribute("aria-pressed", "true")
  })

  test("setting filters surfaces the reset button + clears them on click", async ({
    page,
  }) => {
    await registerEnterprise(page)
    await page.goto("/en/search?type=freelancer")
    await page.waitForLoadState("networkidle")

    await expect(
      page.getByRole("complementary", { name: "Filters" }),
    ).toBeVisible({ timeout: 15_000 })

    // Initially no Reset button (filters are empty)
    await expect(
      page.getByRole("button", { name: "Reset", exact: true }),
    ).not.toBeVisible({ timeout: 1_000 })

    // Set a filter (Now availability pill)
    await page
      .getByRole("complementary", { name: "Filters" })
      .getByRole("button", { name: "Now" })
      .click()

    // Now the reset button appears
    const resetBtn = page.getByRole("button", { name: "Reset", exact: true })
    await expect(resetBtn).toBeVisible()
    await resetBtn.click()

    // After reset, the pill is no longer pressed
    await expect(
      page
        .getByRole("complementary", { name: "Filters" })
        .getByRole("button", { name: "Now" }),
    ).toHaveAttribute("aria-pressed", "false")
  })
})
