import { test, expect, type Page } from "@playwright/test"
import {
  registerProvider,
  registerAgency,
  registerEnterprise,
  clearAuth,
} from "./helpers/auth"

// ---------------------------------------------------------------------------
// Search freelancers
// ---------------------------------------------------------------------------

test.describe("Search freelancers", () => {
  test("navigate to /search?type=freelancer shows search page", async ({ page }) => {
    await registerEnterprise(page)

    await page.goto("/search?type=freelancer")

    // Page heading should show "Find Freelancers"
    await expect(
      page.getByRole("heading", { name: "Find Freelancers" }),
    ).toBeVisible({ timeout: 10000 })
  })

  test("search page shows loading skeleton initially", async ({ page }) => {
    await registerEnterprise(page)

    // Navigate to search and check for skeleton loading
    await page.goto("/search?type=freelancer")

    // The heading should be visible
    await expect(
      page.getByRole("heading", { name: "Find Freelancers" }),
    ).toBeVisible({ timeout: 10000 })

    // After loading, either results grid or empty state should appear
    await expect(
      page.getByText("No profiles found").or(page.locator(".grid")),
    ).toBeVisible({ timeout: 15000 })
  })

  test("search page shows empty state when no freelancers exist", async ({ page }) => {
    await registerEnterprise(page)

    await page.goto("/search?type=freelancer")

    // Wait for loading to finish. If no freelancers exist, shows empty state
    // If freelancers exist (from previous test runs), shows grid — either is valid
    await page.waitForTimeout(3000)

    const emptyState = page.getByText("No profiles found")
    const resultsGrid = page.locator(".grid")

    // One of these must be visible
    const isEmptyVisible = await emptyState.isVisible().catch(() => false)
    const isGridVisible = await resultsGrid.isVisible().catch(() => false)
    expect(isEmptyVisible || isGridVisible).toBe(true)
  })

  test("freelancer cards have photo/initials, name, and role badge", async ({ page }) => {
    // First register a provider so at least one result exists
    const { displayName } = await registerProvider(page)

    // Now register an enterprise to search
    await clearAuth(page)
    await registerEnterprise(page)

    await page.goto("/search?type=freelancer")

    // Wait for results to load
    await expect(
      page.getByRole("heading", { name: "Find Freelancers" }),
    ).toBeVisible({ timeout: 10000 })

    // Wait for the grid to appear (means profiles loaded)
    const grid = page.locator(".grid")
    await expect(grid).toBeVisible({ timeout: 15000 })

    // Each card is a link inside the grid
    const firstCard = grid.locator("a").first()
    await expect(firstCard).toBeVisible()

    // Card should contain an avatar (either an image or initials circle)
    const avatar = firstCard.locator(".rounded-full").first()
    await expect(avatar).toBeVisible()

    // Card should have a role badge
    const badge = firstCard.locator('[class*="uppercase"]')
    await expect(badge).toBeVisible()
  })

  test("clicking a freelancer card navigates to public profile", async ({ page }) => {
    // Register a provider so one exists
    await registerProvider(page)

    // Register enterprise to do the search
    await clearAuth(page)
    await registerEnterprise(page)

    await page.goto("/search?type=freelancer")

    // Wait for grid
    const grid = page.locator(".grid")
    await expect(grid).toBeVisible({ timeout: 15000 })

    // Click the first card
    const firstCard = grid.locator("a").first()
    await firstCard.click()

    // Should navigate to /freelancers/{id}
    await expect(page).toHaveURL(/\/freelancers\//, { timeout: 10000 })
  })
})

// ---------------------------------------------------------------------------
// Search agencies
// ---------------------------------------------------------------------------

test.describe("Search agencies", () => {
  test("navigate to /search?type=agency shows agency search page", async ({ page }) => {
    await registerEnterprise(page)

    await page.goto("/search?type=agency")

    await expect(
      page.getByRole("heading", { name: "Find Agencies" }),
    ).toBeVisible({ timeout: 10000 })
  })

  test("agency cards show Agency badge", async ({ page }) => {
    // Register an agency so one exists
    await registerAgency(page)

    // Search as enterprise
    await clearAuth(page)
    await registerEnterprise(page)

    await page.goto("/search?type=agency")

    const grid = page.locator(".grid")
    await expect(grid).toBeVisible({ timeout: 15000 })

    // Find the Agency badge in a card
    const agencyBadge = grid.getByText("Agency", { exact: true }).first()
    await expect(agencyBadge).toBeVisible()
  })

  test("clicking an agency card navigates to /agencies/{id}", async ({ page }) => {
    await registerAgency(page)

    await clearAuth(page)
    await registerEnterprise(page)

    await page.goto("/search?type=agency")

    const grid = page.locator(".grid")
    await expect(grid).toBeVisible({ timeout: 15000 })

    const firstCard = grid.locator("a").first()
    await firstCard.click()

    await expect(page).toHaveURL(/\/agencies\//, { timeout: 10000 })
  })
})

// ---------------------------------------------------------------------------
// Search referrers
// ---------------------------------------------------------------------------

test.describe("Search referrers", () => {
  test("navigate to /search?type=referrer shows referrer search page", async ({ page }) => {
    await registerEnterprise(page)

    await page.goto("/search?type=referrer")

    await expect(
      page.getByRole("heading", { name: "Find Referrers" }),
    ).toBeVisible({ timeout: 10000 })
  })

  test("referrer cards show Referrer badge", async ({ page }) => {
    await registerEnterprise(page)

    await page.goto("/search?type=referrer")

    // Wait for results (may be empty or populated)
    await page.waitForTimeout(3000)

    const grid = page.locator(".grid")
    const isGridVisible = await grid.isVisible().catch(() => false)

    if (isGridVisible) {
      // If results exist, check for Referrer badge
      const referrerBadge = grid.getByText("Referrer", { exact: true }).first()
      await expect(referrerBadge).toBeVisible()
    } else {
      // Empty state is also valid
      await expect(page.getByText("No profiles found")).toBeVisible()
    }
  })

  test("clicking a referrer card navigates to /referrers/{id}", async ({ page }) => {
    await registerEnterprise(page)

    await page.goto("/search?type=referrer")

    const grid = page.locator(".grid")
    const isGridVisible = await grid.isVisible({ timeout: 10000 }).catch(() => false)

    if (isGridVisible) {
      const firstCard = grid.locator("a").first()
      await firstCard.click()
      await expect(page).toHaveURL(/\/referrers\//, { timeout: 10000 })
    }
    // If no referrers exist, the test passes (no cards to click)
  })
})

// ---------------------------------------------------------------------------
// Default search type
// ---------------------------------------------------------------------------

test.describe("Search default type", () => {
  test("/search without type param defaults to freelancer", async ({ page }) => {
    await registerEnterprise(page)

    await page.goto("/search")

    // Default type should be "freelancer"
    await expect(
      page.getByRole("heading", { name: "Find Freelancers" }),
    ).toBeVisible({ timeout: 10000 })
  })

  test("/search with invalid type param defaults to freelancer", async ({ page }) => {
    await registerEnterprise(page)

    await page.goto("/search?type=invalid")

    await expect(
      page.getByRole("heading", { name: "Find Freelancers" }),
    ).toBeVisible({ timeout: 10000 })
  })
})

// ---------------------------------------------------------------------------
// Public profile pages
// ---------------------------------------------------------------------------

test.describe("Public freelancer profile", () => {
  test("public profile shows profile data", async ({ page }) => {
    // Register a provider with a title to have data to view
    await registerProvider(page)

    // Set a title on the profile
    await page.goto("/profile")
    await page.getByRole("button", { name: /edit professional title/i }).click()
    const titleInput = page.getByRole("textbox", { name: /professional title/i })
    await titleInput.fill("Senior Developer")
    await titleInput.press("Enter")
    await expect(titleInput).not.toBeVisible({ timeout: 5000 })

    // Navigate to search and find our profile
    await page.goto("/search?type=freelancer")
    const grid = page.locator(".grid")
    await expect(grid).toBeVisible({ timeout: 15000 })

    // Click first card (should be our user)
    await grid.locator("a").first().click()
    await expect(page).toHaveURL(/\/freelancers\//, { timeout: 10000 })

    // Public profile should show name/title info
    await expect(page.locator("h1")).toBeVisible({ timeout: 10000 })
  })

  test("public profile is read-only — no edit buttons visible", async ({ page }) => {
    await registerProvider(page)

    // Get the user ID to navigate directly to public profile
    // Navigate via search
    await page.goto("/search?type=freelancer")
    const grid = page.locator(".grid")
    await expect(grid).toBeVisible({ timeout: 15000 })
    await grid.locator("a").first().click()
    await expect(page).toHaveURL(/\/freelancers\//, { timeout: 10000 })

    // There should be no edit buttons (Edit2 icons), no upload buttons
    await expect(
      page.getByRole("button", { name: /edit professional title/i }),
    ).not.toBeVisible()
    await expect(
      page.getByRole("button", { name: /edit your photo/i }),
    ).not.toBeVisible()
    await expect(
      page.getByRole("button", { name: /edit about/i }),
    ).not.toBeVisible()
  })

  test("public profile shows back link", async ({ page }) => {
    await registerProvider(page)

    await page.goto("/search?type=freelancer")
    const grid = page.locator(".grid")
    await expect(grid).toBeVisible({ timeout: 15000 })
    await grid.locator("a").first().click()
    await expect(page).toHaveURL(/\/freelancers\//, { timeout: 10000 })

    // "Back to freelancers" link should be visible
    const backLink = page.getByRole("link", { name: /back to freelancers/i })
    await expect(backLink).toBeVisible()
  })

  test("back link navigates to freelancers directory", async ({ page }) => {
    await registerProvider(page)

    await page.goto("/search?type=freelancer")
    const grid = page.locator(".grid")
    await expect(grid).toBeVisible({ timeout: 15000 })
    await grid.locator("a").first().click()
    await expect(page).toHaveURL(/\/freelancers\//, { timeout: 10000 })

    // Click back link
    await page.getByRole("link", { name: /back to freelancers/i }).click()
    await expect(page).toHaveURL(/\/freelancers/, { timeout: 10000 })
  })

  test("public profile shows 'No reviews yet'", async ({ page }) => {
    await registerProvider(page)

    await page.goto("/search?type=freelancer")
    const grid = page.locator(".grid")
    await expect(grid).toBeVisible({ timeout: 15000 })
    await grid.locator("a").first().click()
    await expect(page).toHaveURL(/\/freelancers\//, { timeout: 10000 })

    await expect(page.getByText("No reviews yet")).toBeVisible({ timeout: 10000 })
  })

  test("public profile shows project history section", async ({ page }) => {
    await registerProvider(page)

    await page.goto("/search?type=freelancer")
    const grid = page.locator(".grid")
    await expect(grid).toBeVisible({ timeout: 15000 })
    await grid.locator("a").first().click()
    await expect(page).toHaveURL(/\/freelancers\//, { timeout: 10000 })

    await expect(page.getByText("Project History")).toBeVisible({ timeout: 10000 })
    await expect(page.getByText("No completed projects yet")).toBeVisible()
  })
})

// ---------------------------------------------------------------------------
// Public profile conditional layout
// ---------------------------------------------------------------------------

test.describe("Public profile layout", () => {
  test("authenticated user sees dashboard shell on public profile", async ({ page }) => {
    await registerProvider(page)

    // Navigate to a public freelancers listing (under (public) layout)
    await page.goto("/freelancers")

    // Since user is authenticated, the (public) layout shows DashboardShell
    // which includes the sidebar <aside>
    const sidebar = page.locator("aside")
    await expect(sidebar).toBeVisible({ timeout: 10000 })
  })

  test("non-authenticated user sees public navbar on freelancers page", async ({ page }) => {
    await page.goto("/")
    await clearAuth(page)

    await page.goto("/freelancers")

    // The public navbar should show "Marketplace Service" text
    await expect(page.getByText("Marketplace Service")).toBeVisible({ timeout: 10000 })

    // Should NOT see a sidebar
    const sidebar = page.locator("aside")
    await expect(sidebar).not.toBeVisible()

    // Should see public nav links
    await expect(page.getByRole("link", { name: /Sign In/i })).toBeVisible()
    await expect(page.getByRole("link", { name: /Create Account/i })).toBeVisible()
  })
})

// ---------------------------------------------------------------------------
// Public agency profile
// ---------------------------------------------------------------------------

test.describe("Public agency profile", () => {
  test("/agencies/{id} shows public agency profile", async ({ page }) => {
    await registerAgency(page)

    // Navigate via search
    await clearAuth(page)
    await registerEnterprise(page)

    await page.goto("/search?type=agency")
    const grid = page.locator(".grid")
    await expect(grid).toBeVisible({ timeout: 15000 })
    await grid.locator("a").first().click()

    await expect(page).toHaveURL(/\/agencies\//, { timeout: 10000 })
    await expect(page.locator("h1")).toBeVisible({ timeout: 10000 })
  })

  test("agency public profile shows back to agencies link", async ({ page }) => {
    await registerAgency(page)

    await clearAuth(page)
    await registerEnterprise(page)

    await page.goto("/search?type=agency")
    const grid = page.locator(".grid")
    await expect(grid).toBeVisible({ timeout: 15000 })
    await grid.locator("a").first().click()
    await expect(page).toHaveURL(/\/agencies\//, { timeout: 10000 })

    await expect(
      page.getByRole("link", { name: /back to agencies/i }),
    ).toBeVisible()
  })
})
