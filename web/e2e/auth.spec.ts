import { test, expect } from "@playwright/test"

test.describe("Authentication", () => {
  test("landing page loads correctly", async ({ page }) => {
    await page.goto("/")
    await expect(page.locator("h1")).toBeVisible()
    await expect(page.locator("text=Commencer gratuitement")).toBeVisible()
  })

  test("can navigate to login page", async ({ page }) => {
    await page.goto("/")
    await page.click("text=Commencer gratuitement")
    await expect(page).toHaveURL(/\/login/)
  })

  test("login page shows form", async ({ page }) => {
    await page.goto("/login")
    await expect(page.locator('input[name="email"]')).toBeVisible()
    await expect(page.locator('input[name="password"]')).toBeVisible()
    await expect(page.locator("text=Se connecter")).toBeVisible()
  })

  test("register page shows form with role selector", async ({ page }) => {
    await page.goto("/register")
    await expect(page.locator('input[name="email"]')).toBeVisible()
    await expect(page.locator('input[name="password"]')).toBeVisible()
    await expect(page.locator('input[name="first_name"]')).toBeVisible()
    await expect(page.locator("text=Agence")).toBeVisible()
    await expect(page.locator("text=Entreprise")).toBeVisible()
    await expect(page.locator("text=Freelance")).toBeVisible()
  })

  test("login with invalid credentials shows error", async ({ page }) => {
    await page.goto("/login")
    await page.fill('input[name="email"]', "wrong@example.com")
    await page.fill('input[name="password"]', "WrongPass123")
    await page.click("text=Se connecter")
    // Should show an error message (stays on login page)
    await expect(page).toHaveURL(/\/login/)
  })

  test("register with valid data redirects to dashboard", async ({ page }) => {
    const uniqueEmail = `test-${Date.now()}@playwright.com`
    await page.goto("/register")
    await page.fill('input[name="email"]', uniqueEmail)
    await page.fill('input[name="password"]', "TestPass123")
    await page.fill('input[name="first_name"]', "Test")
    await page.fill('input[name="last_name"]', "Playwright")
    await page.fill('input[name="display_name"]', "Test Playwright")

    // Select "Freelance" role - click on the role option
    await page.click("text=Freelance")

    await page.click('button[type="submit"]')

    // Should redirect to provider dashboard after successful registration
    await page.waitForURL(/\/dashboard\/provider/, { timeout: 10000 })
  })
})
