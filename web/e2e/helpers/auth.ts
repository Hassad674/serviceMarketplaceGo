import type { Page } from "@playwright/test"

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

export const STRONG_PASSWORD = "TestPass1234!"
export const WEAK_PASSWORD = "weak"

/**
 * Generate a unique email to avoid conflicts between test runs.
 */
export function uniqueEmail(prefix: string): string {
  return `test-${prefix}-${Date.now()}@playwright.com`
}

// ---------------------------------------------------------------------------
// Registration helpers
// ---------------------------------------------------------------------------

export type RegisteredUser = {
  email: string
  password: string
  displayName: string
}

/**
 * Register a provider (freelancer) via the UI.
 * Leaves the browser on /dashboard.
 */
export async function registerProvider(page: Page): Promise<RegisteredUser> {
  const email = uniqueEmail("provider")
  const firstName = "Jean"
  const lastName = `Dupont${Date.now()}`

  // Force English locale so the helper's hard-coded labels match.
  // The site detects FR by default which would otherwise break the test.
  await page.goto("/en/register/provider")
  await page.getByLabel("First name").fill(firstName)
  await page.getByLabel("Last name", { exact: true }).fill(lastName)
  await page.getByLabel("Email").fill(email)
  await page.getByLabel("Password", { exact: true }).fill(STRONG_PASSWORD)
  await page.getByLabel("Confirm password").fill(STRONG_PASSWORD)
  await page.getByRole("button", { name: /Create my freelance account/i }).click()
  await page.waitForURL("**/dashboard", { timeout: 15000 })

  return { email, password: STRONG_PASSWORD, displayName: `${firstName} ${lastName}` }
}

/**
 * Register an agency via the UI.
 * Leaves the browser on /dashboard.
 */
export async function registerAgency(page: Page): Promise<RegisteredUser> {
  const email = uniqueEmail("agency")
  const agencyName = `Agency ${Date.now()}`

  await page.goto("/en/register/agency")
  await page.getByLabel("Agency name").fill(agencyName)
  await page.getByLabel("Email").fill(email)
  await page.getByLabel("Password", { exact: true }).fill(STRONG_PASSWORD)
  await page.getByLabel("Confirm password").fill(STRONG_PASSWORD)
  await page.getByRole("button", { name: /Create my agency account/i }).click()
  await page.waitForURL("**/dashboard", { timeout: 15000 })

  return { email, password: STRONG_PASSWORD, displayName: agencyName }
}

/**
 * Register an enterprise via the UI.
 * Leaves the browser on /dashboard.
 */
export async function registerEnterprise(page: Page): Promise<RegisteredUser> {
  const email = uniqueEmail("enterprise")
  const enterpriseName = `Enterprise ${Date.now()}`

  await page.goto("/en/register/enterprise")
  await page.getByLabel("Company name").fill(enterpriseName)
  await page.getByLabel("Email").fill(email)
  await page.getByLabel("Password", { exact: true }).fill(STRONG_PASSWORD)
  await page.getByLabel("Confirm password").fill(STRONG_PASSWORD)
  await page.getByRole("button", { name: /Create my enterprise account/i }).click()
  await page.waitForURL("**/dashboard", { timeout: 15000 })

  return { email, password: STRONG_PASSWORD, displayName: enterpriseName }
}

// ---------------------------------------------------------------------------
// Login / Logout helpers
// ---------------------------------------------------------------------------

/**
 * Log in via the UI with email + password.
 * Leaves the browser on /dashboard.
 */
export async function login(page: Page, email: string, password: string): Promise<void> {
  await page.goto("/login")
  await page.getByLabel("Email").fill(email)
  await page.getByLabel("Password").fill(password)
  await page.getByRole("button", { name: /Sign In/i }).click()
  await page.waitForURL("**/dashboard", { timeout: 15000 })
}

/**
 * Perform logout from the dashboard.
 * On mobile viewports the hamburger must be opened first.
 * Targets the sidebar logout button inside the <aside> element.
 */
export async function logout(page: Page): Promise<void> {
  // On mobile viewports the sidebar is off-screen behind a hamburger menu.
  const hamburger = page.getByRole("button", { name: "Open menu" })
  if (await hamburger.isVisible().catch(() => false)) {
    await hamburger.click()
    // Wait for sidebar slide-in animation (300ms transition)
    await page.waitForTimeout(350)
  }

  // Click the sidebar logout button (inside <aside>)
  const logoutButton = page.locator("aside").getByRole("button", { name: /Sign Out/i })
  await logoutButton.click()
}

/**
 * Clear all auth cookies and localStorage auth state so the browser
 * appears as an unauthenticated visitor.
 */
export async function clearAuth(page: Page): Promise<void> {
  await page.evaluate(() => {
    // Clear session cookie
    document.cookie = "session_id=; path=/; max-age=0"
    // Legacy cookie
    document.cookie = "access_token=; path=/; max-age=0"
    // Clear workspace cookie
    document.cookie = "workspace=; path=/; max-age=0"
    // Clear localStorage auth stores
    localStorage.removeItem("marketplace-auth")
    localStorage.removeItem("marketplace-theme")
  })
}
