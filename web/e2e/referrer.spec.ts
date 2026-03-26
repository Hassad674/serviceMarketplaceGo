import { test, expect, type Page } from "@playwright/test"
import { registerProvider, registerAgency } from "./helpers/auth"

// ---------------------------------------------------------------------------
// Mode switching
// ---------------------------------------------------------------------------

test.describe("Referrer mode switching", () => {
  test("clicking Business Referrer in sidebar switches to referrer mode", async ({ page }) => {
    await registerProvider(page)

    const sidebar = page.locator("aside")

    // Click the "Business Referrer" button in the sidebar
    const referrerButton = sidebar.getByRole("button", { name: /Business Referrer/i })
    await expect(referrerButton).toBeVisible({ timeout: 10000 })
    await referrerButton.click()

    // After switch, dashboard should show referrer-specific stats
    // Referrer stats include: Referrals, Active Missions, Completed Missions, Commissions
    await expect(page.getByText("Referrals")).toBeVisible({ timeout: 10000 })
    await expect(page.getByText("Commissions")).toBeVisible()
  })

  test("sidebar updates nav items to referrer mode", async ({ page }) => {
    await registerProvider(page)

    const sidebar = page.locator("aside")

    // Switch to referrer mode
    await sidebar.getByRole("button", { name: /Business Referrer/i }).click()

    // After switching, the sidebar should show:
    // - "Freelance Dashboard" button (to switch back)
    // - "Referrer Profile" nav link
    // - "Find Freelancers" nav link
    await expect(
      sidebar.getByRole("button", { name: /Freelance Dashboard/i }),
    ).toBeVisible({ timeout: 10000 })
    await expect(sidebar.getByText("Referrer Profile")).toBeVisible()
    await expect(sidebar.getByText("Find Freelancers")).toBeVisible()
  })

  test("role badge changes to REFERRER in sidebar when in referrer mode", async ({ page }) => {
    await registerProvider(page)

    const sidebar = page.locator("aside")

    // Before switch: should show "Provider"
    await expect(sidebar.getByText("Provider")).toBeVisible({ timeout: 10000 })

    // Switch to referrer mode
    await sidebar.getByRole("button", { name: /Business Referrer/i }).click()

    // After switch: should show "Referrer"
    await expect(sidebar.getByText("Referrer")).toBeVisible({ timeout: 10000 })
  })

  test("clicking Freelance Dashboard switches back to freelance mode", async ({ page }) => {
    await registerProvider(page)

    const sidebar = page.locator("aside")

    // Switch to referrer mode
    await sidebar.getByRole("button", { name: /Business Referrer/i }).click()
    await expect(sidebar.getByText("Referrer")).toBeVisible({ timeout: 10000 })

    // Switch back to freelance mode
    await sidebar.getByRole("button", { name: /Freelance Dashboard/i }).click()

    // Sidebar should revert to freelance nav items
    await expect(sidebar.getByText("My Profile")).toBeVisible({ timeout: 10000 })
    await expect(sidebar.getByText("Provider")).toBeVisible()

    // "Business Referrer" button should be back
    await expect(
      sidebar.getByRole("button", { name: /Business Referrer/i }),
    ).toBeVisible()
  })

  test("sidebar reverts to freelance nav items after switching back", async ({ page }) => {
    await registerProvider(page)

    const sidebar = page.locator("aside")

    // Switch to referrer
    await sidebar.getByRole("button", { name: /Business Referrer/i }).click()
    await expect(sidebar.getByText("Referrer Profile")).toBeVisible({ timeout: 10000 })

    // Switch back
    await sidebar.getByRole("button", { name: /Freelance Dashboard/i }).click()

    // Referrer-specific items should be gone
    await expect(sidebar.getByText("Referrer Profile")).not.toBeVisible()

    // Freelance items should be visible
    await expect(sidebar.getByText("My Profile")).toBeVisible({ timeout: 10000 })
    await expect(sidebar.getByText("Dashboard")).toBeVisible()
  })

  test("dashboard stat cards switch to referrer stats (4 cards)", async ({ page }) => {
    await registerProvider(page)

    // In freelance mode: 3 stat cards
    const statCards = page.locator('[class*="rounded-xl"][class*="border"]').filter({ has: page.locator(".text-2xl") })
    const freelanceCount = await statCards.count()
    expect(freelanceCount).toBe(3)

    // Switch to referrer mode via sidebar
    const sidebar = page.locator("aside")
    await sidebar.getByRole("button", { name: /Business Referrer/i }).click()

    // In referrer mode: 4 stat cards (Referrals, Active Missions, Completed Missions, Commissions)
    await expect(page.getByText("Referrals")).toBeVisible({ timeout: 10000 })
    await expect(page.getByText("Commissions")).toBeVisible()

    const referrerCount = await statCards.count()
    expect(referrerCount).toBe(4)
  })

  test("dashboard has inline referrer switch button for provider", async ({ page }) => {
    await registerProvider(page)

    // In freelance mode, dashboard shows a "Business Referrer" inline button
    const inlineButton = page.locator("main").getByRole("button", { name: /Business Referrer/i })
    await expect(inlineButton).toBeVisible({ timeout: 10000 })
  })

  test("agency user does NOT see referrer switch anywhere", async ({ page }) => {
    await registerAgency(page)

    const sidebar = page.locator("aside")

    // Sidebar should NOT have a Business Referrer button
    await expect(
      sidebar.getByRole("button", { name: /Business Referrer/i }),
    ).not.toBeVisible()

    // Dashboard should NOT have inline referrer switch
    await expect(
      page.locator("main").getByRole("button", { name: /Business Referrer/i }),
    ).not.toBeVisible()
  })
})

// ---------------------------------------------------------------------------
// Referrer mode persistence
// ---------------------------------------------------------------------------

test.describe("Referrer navigation persistence", () => {
  test("referrer mode persists across page navigation", async ({ page }) => {
    await registerProvider(page)

    const sidebar = page.locator("aside")

    // Switch to referrer mode
    await sidebar.getByRole("button", { name: /Business Referrer/i }).click()
    await expect(sidebar.getByText("Referrer")).toBeVisible({ timeout: 10000 })

    // Navigate to search
    await sidebar.getByText("Find Freelancers").click()
    await expect(page).toHaveURL(/\/search/, { timeout: 10000 })

    // Sidebar should still be in referrer mode
    await expect(sidebar.getByText("Referrer")).toBeVisible()
    await expect(
      sidebar.getByRole("button", { name: /Freelance Dashboard/i }),
    ).toBeVisible()
  })

  test("referrer mode persists after page reload via cookie", async ({ page }) => {
    await registerProvider(page)

    const sidebar = page.locator("aside")

    // Switch to referrer mode
    await sidebar.getByRole("button", { name: /Business Referrer/i }).click()
    await expect(sidebar.getByText("Referrer")).toBeVisible({ timeout: 10000 })

    // Verify the workspace cookie is set
    const cookies = await page.context().cookies()
    const workspaceCookie = cookies.find((c) => c.name === "workspace")
    expect(workspaceCookie).toBeDefined()
    expect(workspaceCookie!.value).toBe("referrer")

    // Reload the page
    await page.reload()

    // Wait for hydration
    await page.waitForTimeout(1500)

    // Sidebar should still show referrer mode after reload
    await expect(sidebar.getByText("Referrer")).toBeVisible({ timeout: 10000 })
  })

  test("navigating to /referral auto-switches to referrer mode", async ({ page }) => {
    await registerProvider(page)

    const sidebar = page.locator("aside")

    // Initially in freelance mode
    await expect(sidebar.getByText("Provider")).toBeVisible({ timeout: 10000 })

    // Navigate directly to /referral
    await page.goto("/referral")

    // The useEffect in sidebar auto-syncs workspace to referrer mode
    // when pathname is "/referral"
    await expect(sidebar.getByText("Referrer")).toBeVisible({ timeout: 10000 })
  })
})

// ---------------------------------------------------------------------------
// Referrer profile
// ---------------------------------------------------------------------------

test.describe("Referrer profile", () => {
  test("/referral shows referrer-specific profile page", async ({ page }) => {
    await registerProvider(page)

    await page.goto("/referral")

    // The referrer profile should show a "Business Referrer" badge
    await expect(page.getByText("Business Referrer")).toBeVisible({ timeout: 10000 })
  })

  test("referrer profile shows separate video section", async ({ page }) => {
    await registerProvider(page)

    await page.goto("/referral")

    // The video section should have the referrer-specific title
    await expect(
      page.getByText("Presentation Video — Business Referrer"),
    ).toBeVisible({ timeout: 10000 })
  })

  test("referrer profile shows separate about section", async ({ page }) => {
    await registerProvider(page)

    await page.goto("/referral")

    // The about section should say "About the business referrer"
    await expect(
      page.getByText("About the business referrer"),
    ).toBeVisible({ timeout: 10000 })
  })

  test("referrer profile has video empty state with referrer-specific text", async ({ page }) => {
    await registerProvider(page)

    await page.goto("/referral")

    // Empty video state should reference referrer activity
    await expect(page.getByText("No referrer video")).toBeVisible({ timeout: 10000 })
    await expect(
      page.getByText("Add a video to present your business referrer activity"),
    ).toBeVisible()
  })

  test("referrer about edit saves referrer_about separately from freelance about", async ({ page }) => {
    await registerProvider(page)

    const referrerAbout = `Referrer about text ${Date.now()}`
    const freelanceAbout = `Freelance about text ${Date.now()}`

    // Set referrer about
    await page.goto("/referral")
    await page.getByRole("button", { name: /edit about/i }).click()
    const textarea = page.getByRole("textbox", { name: /about/i })
    await textarea.fill(referrerAbout)
    await page.getByRole("button", { name: /save/i }).click()
    await expect(textarea).not.toBeVisible({ timeout: 10000 })
    await expect(page.getByText(referrerAbout)).toBeVisible()

    // Set freelance about
    await page.goto("/profile")
    await page.getByRole("button", { name: /edit about/i }).click()
    const freelanceTextarea = page.getByRole("textbox", { name: /about/i })
    await freelanceTextarea.fill(freelanceAbout)
    await page.getByRole("button", { name: /save/i }).click()
    await expect(freelanceTextarea).not.toBeVisible({ timeout: 10000 })
    await expect(page.getByText(freelanceAbout)).toBeVisible()

    // Verify they are independent: go back to referrer profile
    await page.goto("/referral")
    await expect(page.getByText(referrerAbout)).toBeVisible({ timeout: 10000 })
    // The freelance about should NOT appear on referrer page
    await expect(page.getByText(freelanceAbout)).not.toBeVisible()

    // And vice versa
    await page.goto("/profile")
    await expect(page.getByText(freelanceAbout)).toBeVisible({ timeout: 10000 })
    await expect(page.getByText(referrerAbout)).not.toBeVisible()
  })

  test("non-provider user on /referral sees 'provider only' message", async ({ page }) => {
    await registerAgency(page)

    await page.goto("/referral")

    // Should show the restricted message
    await expect(
      page.getByText("This page is only available for provider accounts."),
    ).toBeVisible({ timeout: 10000 })
  })

  test("referrer profile video upload modal opens", async ({ page }) => {
    await registerProvider(page)

    await page.goto("/referral")

    // Click "Add a video" button
    await page.getByRole("button", { name: "Add a video" }).click()

    const modal = page.getByRole("dialog")
    await expect(modal).toBeVisible({ timeout: 5000 })
    await expect(modal).toContainText("Add a video")
  })
})

// ---------------------------------------------------------------------------
// Referrer mode header integration
// ---------------------------------------------------------------------------

test.describe("Referrer header integration", () => {
  test("header dropdown shows REFERRER badge when in referrer mode", async ({ page }) => {
    await registerProvider(page)

    const sidebar = page.locator("aside")

    // Switch to referrer mode
    await sidebar.getByRole("button", { name: /Business Referrer/i }).click()
    await expect(sidebar.getByText("Referrer")).toBeVisible({ timeout: 10000 })

    // Open header dropdown
    const header = page.locator("header")
    const dropdownTrigger = header.locator("button").filter({ has: page.locator(".rounded-full") })
    await dropdownTrigger.click()

    // Dropdown should show the REFERRER badge
    await expect(
      header.locator('[class*="uppercase"]').filter({ hasText: /Referrer/i }),
    ).toBeVisible()
  })

  test("header dropdown profile link goes to /referral when in referrer mode", async ({ page }) => {
    await registerProvider(page)

    const sidebar = page.locator("aside")

    // Switch to referrer mode
    await sidebar.getByRole("button", { name: /Business Referrer/i }).click()
    await expect(sidebar.getByText("Referrer")).toBeVisible({ timeout: 10000 })

    // Open header dropdown
    const header = page.locator("header")
    const dropdownTrigger = header.locator("button").filter({ has: page.locator(".rounded-full") })
    await dropdownTrigger.click()

    // Click "My Profile" in dropdown — should go to /referral in referrer mode
    await page.getByRole("link", { name: /My Profile/i }).click()
    await expect(page).toHaveURL(/\/referral/, { timeout: 10000 })
  })
})
