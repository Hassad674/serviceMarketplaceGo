import { test, expect, type Page } from "@playwright/test"
import {
  registerProvider,
  registerAgency,
  registerEnterprise,
  clearAuth,
  STRONG_PASSWORD,
  uniqueEmail,
} from "./helpers/auth"

// ---------------------------------------------------------------------------
// Provider dashboard
// ---------------------------------------------------------------------------

test.describe("Provider dashboard", () => {
  test("shows welcome banner with user first name", async ({ page }) => {
    const { displayName } = await registerProvider(page)
    const firstName = displayName.split(" ")[0]

    // The dashboard banner shows "Welcome back, {name}"
    const welcomeBanner = page.locator("h1")
    await expect(welcomeBanner).toContainText(firstName, { timeout: 10000 })
  })

  test("shows 3 stat cards for provider", async ({ page }) => {
    await registerProvider(page)

    // Provider stats: Active Missions, Unread Messages, Monthly Revenue
    await expect(page.getByText("Active Missions")).toBeVisible({ timeout: 10000 })
    await expect(page.getByText("Unread Messages")).toBeVisible()
    await expect(page.getByText("Monthly Revenue")).toBeVisible()
  })

  test("sidebar shows correct provider nav items", async ({ page }) => {
    await registerProvider(page)

    const sidebar = page.locator("aside")

    // Provider should see: Dashboard, My Profile
    await expect(sidebar.getByText("Dashboard")).toBeVisible({ timeout: 10000 })
    await expect(sidebar.getByText("My Profile")).toBeVisible()
  })

  test("sidebar shows Business Referrer switch button", async ({ page }) => {
    await registerProvider(page)

    const sidebar = page.locator("aside")

    // Provider should see the referrer mode switch
    await expect(
      sidebar.getByRole("button", { name: /Business Referrer/i }),
    ).toBeVisible({ timeout: 10000 })
  })

  test("provider role badge shows PROVIDER in sidebar", async ({ page }) => {
    await registerProvider(page)

    const sidebar = page.locator("aside")
    await expect(sidebar.getByText("Provider")).toBeVisible({ timeout: 10000 })
  })
})

// ---------------------------------------------------------------------------
// Agency dashboard
// ---------------------------------------------------------------------------

test.describe("Agency dashboard", () => {
  test("shows welcome banner with agency name", async ({ page }) => {
    const { displayName } = await registerAgency(page)

    const welcomeBanner = page.locator("h1")
    await expect(welcomeBanner).toContainText(displayName, { timeout: 10000 })
  })

  test("shows 3 stat cards for agency", async ({ page }) => {
    await registerAgency(page)

    // Agency stats: Active Missions, Unread Messages, Monthly Revenue
    await expect(page.getByText("Active Missions")).toBeVisible({ timeout: 10000 })
    await expect(page.getByText("Unread Messages")).toBeVisible()
    await expect(page.getByText("Monthly Revenue")).toBeVisible()
  })

  test("sidebar shows correct agency nav items", async ({ page }) => {
    await registerAgency(page)

    const sidebar = page.locator("aside")

    // Agency should see: Dashboard, My Profile, Find Freelancers, Find Referrers
    await expect(sidebar.getByText("Dashboard")).toBeVisible({ timeout: 10000 })
    await expect(sidebar.getByText("My Profile")).toBeVisible()
    await expect(sidebar.getByText("Find Freelancers")).toBeVisible()
    await expect(sidebar.getByText("Find Referrers")).toBeVisible()
  })

  test("sidebar does NOT show Business Referrer switch for agency", async ({ page }) => {
    await registerAgency(page)

    const sidebar = page.locator("aside")

    // Business Referrer switch is only for providers
    await expect(
      sidebar.getByRole("button", { name: /Business Referrer/i }),
    ).not.toBeVisible()
  })

  test("agency role badge shows AGENCY in sidebar", async ({ page }) => {
    await registerAgency(page)

    const sidebar = page.locator("aside")
    await expect(sidebar.getByText("Agency")).toBeVisible({ timeout: 10000 })
  })
})

// ---------------------------------------------------------------------------
// Enterprise dashboard
// ---------------------------------------------------------------------------

test.describe("Enterprise dashboard", () => {
  test("shows welcome banner with enterprise name", async ({ page }) => {
    const { displayName } = await registerEnterprise(page)

    const welcomeBanner = page.locator("h1")
    await expect(welcomeBanner).toContainText(displayName, { timeout: 10000 })
  })

  test("shows 3 stat cards for enterprise", async ({ page }) => {
    await registerEnterprise(page)

    // Enterprise stats: Active Projects, Unread Messages, Total Budget
    await expect(page.getByText("Active Projects")).toBeVisible({ timeout: 10000 })
    await expect(page.getByText("Unread Messages")).toBeVisible()
    await expect(page.getByText("Total Budget")).toBeVisible()
  })

  test("sidebar shows correct enterprise nav items", async ({ page }) => {
    await registerEnterprise(page)

    const sidebar = page.locator("aside")

    // Enterprise should see: Dashboard, Find Freelancers, Find Agencies, Find Referrers
    await expect(sidebar.getByText("Dashboard")).toBeVisible({ timeout: 10000 })
    await expect(sidebar.getByText("Find Freelancers")).toBeVisible()
    await expect(sidebar.getByText("Find Agencies")).toBeVisible()
    await expect(sidebar.getByText("Find Referrers")).toBeVisible()
  })

  test("sidebar does NOT show Business Referrer switch for enterprise", async ({ page }) => {
    await registerEnterprise(page)

    const sidebar = page.locator("aside")

    await expect(
      sidebar.getByRole("button", { name: /Business Referrer/i }),
    ).not.toBeVisible()
  })

  test("enterprise role badge shows ENTERPRISE in sidebar", async ({ page }) => {
    await registerEnterprise(page)

    const sidebar = page.locator("aside")
    await expect(sidebar.getByText("Enterprise")).toBeVisible({ timeout: 10000 })
  })
})

// ---------------------------------------------------------------------------
// Dashboard home redirect
// ---------------------------------------------------------------------------

test.describe("Dashboard home redirect", () => {
  test("authenticated user on / gets redirected to /dashboard", async ({ page }) => {
    await registerProvider(page)

    // Navigate to the landing page
    await page.goto("/")

    // Middleware should detect session_id cookie and redirect to /dashboard
    await expect(page).toHaveURL(/\/dashboard/, { timeout: 10000 })
  })

  test("unauthenticated user stays on / (landing page)", async ({ page }) => {
    await page.goto("/")
    await clearAuth(page)

    await page.goto("/")
    await expect(page).toHaveURL(/^\/$|\/en$/)
    await expect(page.locator("h1")).toBeVisible()
  })
})

// ---------------------------------------------------------------------------
// Dashboard stat card interactions
// ---------------------------------------------------------------------------

test.describe("Dashboard stat cards", () => {
  test("stat cards show zero values for new provider account", async ({ page }) => {
    await registerProvider(page)

    // All stat values should show "0" or "0 EUR" for a fresh account
    const statValues = page.locator(".text-2xl.font-bold")
    const count = await statValues.count()
    expect(count).toBeGreaterThanOrEqual(3)

    // Each value should contain "0"
    for (let i = 0; i < count; i++) {
      const text = await statValues.nth(i).textContent()
      expect(text).toMatch(/^0/)
    }
  })

  test("stat cards show zero values for new enterprise account", async ({ page }) => {
    await registerEnterprise(page)

    const statValues = page.locator(".text-2xl.font-bold")
    const count = await statValues.count()
    expect(count).toBeGreaterThanOrEqual(3)

    for (let i = 0; i < count; i++) {
      const text = await statValues.nth(i).textContent()
      expect(text).toMatch(/^0/)
    }
  })
})
