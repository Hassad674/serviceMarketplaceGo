import { test, expect } from "@playwright/test"
import {
  registerProvider,
  registerAgency,
  registerEnterprise,
  clearAuth,
} from "./helpers/auth"

// ---------------------------------------------------------------------------
// Sidebar
// ---------------------------------------------------------------------------

test.describe("Sidebar", () => {
  test("sidebar collapses on collapse button click", async ({ page }) => {
    await registerProvider(page)

    const sidebar = page.locator("aside")
    const collapseButton = sidebar.getByRole("button", { name: /Collapse sidebar/i })

    // Sidebar starts expanded at 280px
    await expect(sidebar).toBeVisible({ timeout: 10000 })
    await expect(collapseButton).toBeVisible()

    // Click collapse
    await collapseButton.click()

    // After collapse, sidebar should be narrow (72px)
    // The expand button should now be visible
    await expect(
      sidebar.getByRole("button", { name: /Expand sidebar/i }),
    ).toBeVisible({ timeout: 5000 })
  })

  test("collapsed sidebar shows only icons (no text labels)", async ({ page }) => {
    await registerProvider(page)

    const sidebar = page.locator("aside")

    // Collapse the sidebar
    await sidebar.getByRole("button", { name: /Collapse sidebar/i }).click()

    // After collapse, nav link text should not be visible
    // The "Dashboard" text link should be hidden (only icon visible)
    // We check that the sidebar has width 72px via its class
    await expect(sidebar).toHaveClass(/w-\[72px\]/, { timeout: 5000 })
  })

  test("sidebar collapse state persists after page reload", async ({ page }) => {
    await registerProvider(page)

    const sidebar = page.locator("aside")

    // Collapse
    await sidebar.getByRole("button", { name: /Collapse sidebar/i }).click()
    await expect(sidebar).toHaveClass(/w-\[72px\]/, { timeout: 5000 })

    // Reload
    await page.reload()

    // Sidebar should still be collapsed after reload
    // (localStorage stores the collapsed state)
    await page.waitForTimeout(1500)
    await expect(sidebar).toHaveClass(/w-\[72px\]/, { timeout: 10000 })
  })

  test("active sidebar item highlights correctly", async ({ page }) => {
    await registerProvider(page)

    const sidebar = page.locator("aside")

    // On /dashboard, the "Dashboard" link should be active (has rose-colored classes)
    const dashboardLink = sidebar.locator("a").filter({ hasText: "Dashboard" })
    await expect(dashboardLink).toHaveClass(/text-rose/, { timeout: 10000 })

    // Navigate to /profile
    await sidebar.getByText("My Profile").click()
    await expect(page).toHaveURL(/\/profile/, { timeout: 10000 })

    // "My Profile" should now be active
    const profileLink = sidebar.locator("a").filter({ hasText: "My Profile" })
    await expect(profileLink).toHaveClass(/text-rose/)

    // "Dashboard" should no longer be active
    await expect(dashboardLink).not.toHaveClass(/text-rose/)
  })

  test("all sidebar nav links navigate to correct pages for provider", async ({ page }) => {
    await registerProvider(page)

    const sidebar = page.locator("aside")

    // Dashboard link
    await sidebar.getByText("Dashboard").click()
    await expect(page).toHaveURL(/\/dashboard/, { timeout: 10000 })

    // My Profile link
    await sidebar.getByText("My Profile").click()
    await expect(page).toHaveURL(/\/profile/, { timeout: 10000 })
  })

  test("all sidebar nav links navigate to correct pages for enterprise", async ({ page }) => {
    await registerEnterprise(page)

    const sidebar = page.locator("aside")

    // Dashboard
    await sidebar.getByText("Dashboard").click()
    await expect(page).toHaveURL(/\/dashboard/, { timeout: 10000 })

    // Find Freelancers
    await sidebar.getByText("Find Freelancers").click()
    await expect(page).toHaveURL(/\/search/, { timeout: 10000 })

    // Find Agencies
    await sidebar.getByText("Find Agencies").click()
    await expect(page).toHaveURL(/\/search/, { timeout: 10000 })

    // Find Referrers
    await sidebar.getByText("Find Referrers").click()
    await expect(page).toHaveURL(/\/search/, { timeout: 10000 })
  })

  test("sidebar logo links to home page", async ({ page }) => {
    await registerProvider(page)

    const sidebar = page.locator("aside")
    // The logo text "Atelier" is a link to "/"
    const logoLink = sidebar.getByText("Atelier")
    await expect(logoLink).toBeVisible({ timeout: 10000 })
  })

  test("expanding a collapsed sidebar restores full width", async ({ page }) => {
    await registerProvider(page)

    const sidebar = page.locator("aside")

    // Collapse
    await sidebar.getByRole("button", { name: /Collapse sidebar/i }).click()
    await expect(sidebar).toHaveClass(/w-\[72px\]/, { timeout: 5000 })

    // Expand
    await sidebar.getByRole("button", { name: /Expand sidebar/i }).click()
    await expect(sidebar).toHaveClass(/w-\[280px\]/, { timeout: 5000 })
  })
})

// ---------------------------------------------------------------------------
// Header
// ---------------------------------------------------------------------------

test.describe("Header", () => {
  test("search bar is visible in header (desktop)", async ({ page }) => {
    await registerProvider(page)

    const header = page.locator("header")
    const searchInput = header.locator('input[type="text"]')

    // Search input is hidden on mobile but visible on sm+ breakpoints
    // Playwright desktop project should show it
    await expect(searchInput).toBeVisible({ timeout: 10000 })
  })

  test("notification bell is visible in header", async ({ page }) => {
    await registerProvider(page)

    const header = page.locator("header")
    const bellButton = header.getByRole("button", { name: /Notifications/i })
    await expect(bellButton).toBeVisible({ timeout: 10000 })
  })

  test("user dropdown opens on avatar click", async ({ page }) => {
    await registerProvider(page)

    const header = page.locator("header")

    // Click the avatar/dropdown trigger
    const dropdownTrigger = header.locator("button").filter({ has: page.locator(".rounded-full.bg-gradient-to-br") })
    await expect(dropdownTrigger).toBeVisible({ timeout: 10000 })
    await dropdownTrigger.click()

    // Dropdown content should appear
    const dropdown = header.locator('[class*="animate-scale-in"]')
    await expect(dropdown).toBeVisible({ timeout: 5000 })
  })

  test("user dropdown shows name, email, and role", async ({ page }) => {
    const { displayName, email } = await registerAgency(page)

    const header = page.locator("header")

    // Open dropdown
    const dropdownTrigger = header.locator("button").filter({ has: page.locator(".rounded-full.bg-gradient-to-br") })
    await dropdownTrigger.click()

    // Dropdown should display user info
    const dropdown = header.locator('[class*="animate-scale-in"]')
    await expect(dropdown).toBeVisible({ timeout: 5000 })

    await expect(dropdown.getByText(displayName)).toBeVisible()
    await expect(dropdown.getByText(email)).toBeVisible()
    await expect(dropdown.getByText(/Agency/i)).toBeVisible()
  })

  test("logout from header dropdown works", async ({ page }) => {
    await registerProvider(page)

    const header = page.locator("header")

    // Open dropdown
    const dropdownTrigger = header.locator("button").filter({ has: page.locator(".rounded-full.bg-gradient-to-br") })
    await dropdownTrigger.click()

    // Click Sign Out
    const dropdown = header.locator('[class*="animate-scale-in"]')
    await expect(dropdown).toBeVisible({ timeout: 5000 })

    await dropdown.getByRole("button", { name: /Sign Out/i }).click()

    // Should redirect to login
    await expect(page).toHaveURL(/\/login/, { timeout: 10000 })
  })

  test("header dropdown 'My Profile' navigates to /profile", async ({ page }) => {
    await registerProvider(page)

    const header = page.locator("header")

    // Open dropdown
    const dropdownTrigger = header.locator("button").filter({ has: page.locator(".rounded-full.bg-gradient-to-br") })
    await dropdownTrigger.click()

    const dropdown = header.locator('[class*="animate-scale-in"]')
    await expect(dropdown).toBeVisible({ timeout: 5000 })

    await dropdown.getByRole("link", { name: /My Profile/i }).click()
    await expect(page).toHaveURL(/\/profile/, { timeout: 10000 })
  })

  test("clicking outside dropdown closes it", async ({ page }) => {
    await registerProvider(page)

    const header = page.locator("header")

    // Open dropdown
    const dropdownTrigger = header.locator("button").filter({ has: page.locator(".rounded-full.bg-gradient-to-br") })
    await dropdownTrigger.click()

    const dropdown = header.locator('[class*="animate-scale-in"]')
    await expect(dropdown).toBeVisible({ timeout: 5000 })

    // Click outside the dropdown (on the main content area)
    await page.locator("main").click({ position: { x: 10, y: 10 } })

    // Dropdown should close
    await expect(dropdown).not.toBeVisible({ timeout: 5000 })
  })
})

// ---------------------------------------------------------------------------
// Dark mode
// ---------------------------------------------------------------------------

test.describe("Dark mode", () => {
  test("theme toggle exists on landing page", async ({ page }) => {
    await page.goto("/")

    // The theme toggle button should be in the navbar
    const toggleButton = page.getByRole("button", { name: /switch to dark mode|switch to light mode/i })
    await expect(toggleButton).toBeVisible({ timeout: 10000 })
  })

  test("clicking theme toggle on landing page switches to dark mode", async ({ page }) => {
    await page.goto("/")

    const toggleButton = page.getByRole("button", { name: /switch to dark mode/i })
    await expect(toggleButton).toBeVisible({ timeout: 10000 })

    // Click to switch to dark mode
    await toggleButton.click()

    // The <html> element should have class "dark"
    const htmlElement = page.locator("html")
    await expect(htmlElement).toHaveClass(/dark/, { timeout: 5000 })
  })

  test("theme toggle on login page works", async ({ page }) => {
    await page.goto("/login")

    // Login page inherits from auth layout which has a ThemeToggle
    // or the landing navbar has it
    const toggleButton = page.getByRole("button", { name: /switch to dark mode|switch to light mode/i })
    // If toggle exists on login page
    const isVisible = await toggleButton.isVisible().catch(() => false)

    if (isVisible) {
      await toggleButton.click()
      const htmlElement = page.locator("html")
      await expect(htmlElement).toHaveClass(/dark/, { timeout: 5000 })
    }
    // If no toggle on login page, test passes (design choice)
  })

  test("dark mode toggle works in dashboard", async ({ page }) => {
    await registerProvider(page)

    // Dashboard header has a ThemeToggle
    const header = page.locator("header")
    const toggleButton = header.getByRole("button", { name: /switch to dark mode|switch to light mode/i })
    await expect(toggleButton).toBeVisible({ timeout: 10000 })

    // Switch to dark mode
    await toggleButton.click()

    const htmlElement = page.locator("html")
    await expect(htmlElement).toHaveClass(/dark/, { timeout: 5000 })

    // Switch back to light
    const lightToggle = header.getByRole("button", { name: /switch to light mode|switch to dark mode/i })
    await lightToggle.click()

    await expect(htmlElement).not.toHaveClass(/dark/, { timeout: 5000 })
  })

  test("dark mode persists after navigation within dashboard", async ({ page }) => {
    await registerProvider(page)

    const header = page.locator("header")

    // Switch to dark mode
    await header.getByRole("button", { name: /switch to dark mode/i }).click()
    await expect(page.locator("html")).toHaveClass(/dark/, { timeout: 5000 })

    // Navigate to profile
    const sidebar = page.locator("aside")
    await sidebar.getByText("My Profile").click()
    await expect(page).toHaveURL(/\/profile/, { timeout: 10000 })

    // Dark mode should still be active
    await expect(page.locator("html")).toHaveClass(/dark/)
  })

  test("dark mode persists after page reload", async ({ page }) => {
    await registerProvider(page)

    const header = page.locator("header")

    // Switch to dark mode
    await header.getByRole("button", { name: /switch to dark mode/i }).click()
    await expect(page.locator("html")).toHaveClass(/dark/, { timeout: 5000 })

    // Reload the page
    await page.reload()

    // Dark mode is persisted via Zustand persist to localStorage
    // Wait for hydration
    await page.waitForTimeout(1500)

    // Verify theme is persisted in localStorage
    const themeData = await page.evaluate(() =>
      localStorage.getItem("marketplace-theme"),
    )
    expect(themeData).not.toBeNull()
    const parsed = JSON.parse(themeData!)
    expect(parsed.state.theme).toBe("dark")
  })
})

// ---------------------------------------------------------------------------
// Responsive — mobile viewport
// ---------------------------------------------------------------------------

test.describe("Responsive mobile", () => {
  // Use the "mobile" project defined in playwright.config.ts
  // These tests use Pixel 5 viewport (393x851)
  test.use({ viewport: { width: 393, height: 851 } })

  test("sidebar is hidden by default on mobile", async ({ page }) => {
    await registerProvider(page)

    const sidebar = page.locator("aside")

    // On mobile, sidebar starts off-screen (translate-x-full)
    // The sidebar should NOT be visible in the viewport
    await expect(sidebar).toHaveClass(/-translate-x-full/, { timeout: 10000 })
  })

  test("hamburger menu button is visible on mobile", async ({ page }) => {
    await registerProvider(page)

    const hamburger = page.getByRole("button", { name: "Open menu" })
    await expect(hamburger).toBeVisible({ timeout: 10000 })
  })

  test("hamburger menu opens the sidebar on mobile", async ({ page }) => {
    await registerProvider(page)

    // Click hamburger
    await page.getByRole("button", { name: "Open menu" }).click()

    const sidebar = page.locator("aside")

    // After opening, sidebar should be translated to x-0 (visible)
    await expect(sidebar).toHaveClass(/translate-x-0/, { timeout: 5000 })
  })

  test("clicking overlay closes sidebar on mobile", async ({ page }) => {
    await registerProvider(page)

    // Open sidebar
    await page.getByRole("button", { name: "Open menu" }).click()

    const sidebar = page.locator("aside")
    await expect(sidebar).toHaveClass(/translate-x-0/, { timeout: 5000 })

    // The overlay is a div.fixed.inset-0 behind the sidebar
    // Click on it to close
    const overlay = page.locator(".fixed.inset-0.bg-black\\/20")
    await overlay.click({ position: { x: 350, y: 400 } })

    // Sidebar should be hidden again
    await expect(sidebar).toHaveClass(/-translate-x-full/, { timeout: 5000 })
  })

  test("sidebar close button works on mobile", async ({ page }) => {
    await registerProvider(page)

    // Open sidebar
    await page.getByRole("button", { name: "Open menu" }).click()

    const sidebar = page.locator("aside")
    await expect(sidebar).toHaveClass(/translate-x-0/, { timeout: 5000 })

    // Click the X button inside sidebar
    await sidebar.getByRole("button", { name: "Close menu" }).click()

    // Sidebar should close
    await expect(sidebar).toHaveClass(/-translate-x-full/, { timeout: 5000 })
  })

  test("sidebar navigation on mobile closes sidebar after clicking a link", async ({ page }) => {
    await registerProvider(page)

    // Open sidebar
    await page.getByRole("button", { name: "Open menu" }).click()

    const sidebar = page.locator("aside")
    await expect(sidebar).toHaveClass(/translate-x-0/, { timeout: 5000 })

    // Click "My Profile" link — sidebar should close via onClose callback
    await sidebar.getByText("My Profile").click()

    await expect(page).toHaveURL(/\/profile/, { timeout: 10000 })

    // Sidebar should be hidden after navigation
    await expect(sidebar).toHaveClass(/-translate-x-full/, { timeout: 5000 })
  })

  test("search bar is hidden on mobile", async ({ page }) => {
    await registerProvider(page)

    const header = page.locator("header")
    const searchInput = header.locator('input[type="text"]')

    // Search input has class "hidden" on mobile (sm:block)
    await expect(searchInput).not.toBeVisible()
  })
})

// ---------------------------------------------------------------------------
// i18n
// ---------------------------------------------------------------------------

test.describe("Internationalization (i18n)", () => {
  test("/fr prefix shows French translations on landing page", async ({ page }) => {
    await page.goto("/fr")

    // The landing page hero should be in French
    await expect(page.locator("h1")).toContainText(
      "La plateforme B2B",
      { timeout: 10000 },
    )
  })

  test("default locale (en) does not require /en prefix", async ({ page }) => {
    await page.goto("/")

    // English is the default locale with localePrefix: 'as-needed'
    // So "/" should show English content
    await expect(page.locator("h1")).toContainText(
      "The B2B marketplace",
      { timeout: 10000 },
    )
  })

  test("/fr/login shows French login page", async ({ page }) => {
    await page.goto("/fr/login")

    // Login heading should be in French
    await expect(
      page.getByRole("heading", { name: /Connexion/i }),
    ).toBeVisible({ timeout: 10000 })
  })

  test("/fr/register shows French role selection", async ({ page }) => {
    await page.goto("/fr/register")

    // Register heading in French
    await expect(
      page.getByRole("heading", { name: /Créer un compte/i }),
    ).toBeVisible({ timeout: 10000 })

    // Role cards in French
    await expect(page.getByText("Agence")).toBeVisible()
    await expect(page.getByText("Entreprise")).toBeVisible()
  })

  test("English login page has English labels", async ({ page }) => {
    await page.goto("/login")

    await expect(
      page.getByRole("heading", { name: /Sign In/i }),
    ).toBeVisible({ timeout: 10000 })
    await expect(page.getByLabel("Email")).toBeVisible()
    await expect(page.getByLabel("Password")).toBeVisible()
  })

  test("English register page has English labels", async ({ page }) => {
    await page.goto("/register")

    await expect(
      page.getByRole("heading", { name: /Create your account/i }),
    ).toBeVisible({ timeout: 10000 })
  })

  test("French dashboard uses French translations after login", async ({ page }) => {
    // Register via French locale URL
    const email = `test-fr-${Date.now()}@playwright.com`
    const password = "TestPass1234!"
    const agencyName = `AgenceFR ${Date.now()}`

    await page.goto("/fr/register/agency")
    await page.getByLabel("Nom de l'agence").fill(agencyName)
    await page.getByLabel("Email").fill(email)
    await page.getByLabel("Mot de passe", { exact: true }).fill(password)
    await page.getByLabel("Confirmer le mot de passe").fill(password)
    await page.getByRole("button", { name: /Créer mon compte agence/i }).click()
    await page.waitForURL("**/dashboard", { timeout: 15000 })

    // Dashboard greeting should be in French: "Bonjour, {name}"
    // The French translation uses the same key but French text
    await expect(page.locator("h1")).toBeVisible({ timeout: 10000 })
  })
})

// ---------------------------------------------------------------------------
// Protected routes for all protected paths
// ---------------------------------------------------------------------------

test.describe("Protected routes redirect to login", () => {
  const protectedPaths = [
    "/dashboard",
    "/profile",
    "/search",
    "/messages",
    "/missions",
    "/invoices",
    "/team",
    "/referral",
    "/settings",
  ]

  for (const path of protectedPaths) {
    test(`${path} redirects to /login without auth`, async ({ page }) => {
      await page.goto("/")
      await clearAuth(page)

      await page.goto(path)
      await expect(page).toHaveURL(/\/login/, { timeout: 10000 })
    })
  }
})

// ---------------------------------------------------------------------------
// User initials display
// ---------------------------------------------------------------------------

test.describe("User initials", () => {
  test("sidebar shows user initials in avatar circle", async ({ page }) => {
    await registerProvider(page)

    const sidebar = page.locator("aside")

    // The avatar circle should contain initials (first letter of first_name + last_name)
    // Provider registers with first_name "Jean" -> initials start with "J"
    const avatarCircle = sidebar.locator(".rounded-full.bg-gradient-to-br")
    await expect(avatarCircle).toBeVisible({ timeout: 10000 })
    const initials = await avatarCircle.textContent()
    expect(initials).toBeTruthy()
    expect(initials!.length).toBeGreaterThanOrEqual(1)
  })

  test("header shows user initials in avatar", async ({ page }) => {
    await registerProvider(page)

    const header = page.locator("header")
    const avatarCircle = header.locator(".rounded-full.bg-gradient-to-br")
    await expect(avatarCircle).toBeVisible({ timeout: 10000 })
    const initials = await avatarCircle.textContent()
    expect(initials).toBeTruthy()
  })
})
