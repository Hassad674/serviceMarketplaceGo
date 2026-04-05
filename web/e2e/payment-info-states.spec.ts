import { test, expect, type Page, type Route } from "@playwright/test"

// ---------------------------------------------------------------------------
// /payment-info — visual state matrix
// ---------------------------------------------------------------------------
//
// The page auto-detects 4 modes based on the /account-status endpoint:
//   - loading     → initial fetch in flight
//   - wizard      → no account yet (404 on status)
//   - onboarding  → details_submitted=false, show AccountOnboarding
//   - dashboard   → details_submitted=true, show AccountManagement
//
// These tests mock every account-status payload shape the backend can emit
// and assert the page renders the right visual state. They do NOT test the
// Stripe iframe content (cross-origin, covered elsewhere).
//
// Value: proves the marketplace correctly surfaces every Stripe lifecycle
// scenario — missing info, extra requirements, suspension, activation —
// without the user ever having to visit stripe.com.
// ---------------------------------------------------------------------------

test.describe.configure({ mode: "parallel" })

type StatusResponse = {
  account_id: string
  country: string
  business_type: string
  charges_enabled: boolean
  payouts_enabled: boolean
  details_submitted: boolean
  requirements_currently_due: string[]
  requirements_past_due: string[]
  requirements_eventually_due: string[]
  requirements_pending_verification: string[]
  requirements_count: number
  disabled_reason?: string
}

const ACCOUNT_ACTIVE_CLEAN: StatusResponse = {
  account_id: "acct_clean",
  country: "FR",
  business_type: "individual",
  charges_enabled: true,
  payouts_enabled: true,
  details_submitted: true,
  requirements_currently_due: [],
  requirements_past_due: [],
  requirements_eventually_due: [],
  requirements_pending_verification: [],
  requirements_count: 0,
}

async function mockRoutes(page: Page, status: StatusResponse | null) {
  // /auth/me returns 401 — page still loads via layout-level handling
  await page.route(/\/api\/v1\/auth\/me/, async (route: Route) => {
    await route.fulfill({
      status: 401,
      contentType: "application/json",
      body: JSON.stringify({ error: { code: "unauthorized" } }),
    })
  })

  // /account-status
  await page.route(/\/api\/v1\/payment-info\/account-status/, async (route: Route) => {
    if (status === null) {
      await route.fulfill({
        status: 404,
        contentType: "application/json",
        body: JSON.stringify({ error: { code: "no_account" } }),
      })
      return
    }
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify(status),
    })
  })

  // /account-session (may not be called but mock it anyway)
  await page.route(/\/api\/v1\/payment-info\/account-session/, async (route: Route) => {
    if (route.request().method() === "DELETE") {
      await route.fulfill({ status: 204, body: "" })
      return
    }
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        client_secret: "test_fake_secret",
        account_id: "acct_fake",
        expires_at: Math.floor(Date.now() / 1000) + 3600,
      }),
    })
  })
}

async function gotoPage(page: Page) {
  await page.goto("/fr/payment-info", { waitUntil: "domcontentloaded" })
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

test.describe("payment-info — visual state matrix", () => {
  test("no account (404) → shows wizard with country selector", async ({ page }) => {
    await mockRoutes(page, null)
    await gotoPage(page)

    await expect(
      page.getByRole("heading", { name: /Configurons vos paiements/i }),
    ).toBeVisible({ timeout: 10_000 })

    // Wizard CTA disabled until both country and business_type are picked
    const continueBtn = page.getByRole("button", { name: /Continuer/i })
    await expect(continueBtn).toBeVisible()
    await expect(continueBtn).toBeDisabled()
  })

  test("active + 0 requirements → shows green 'Compte entièrement actif'", async ({
    page,
  }) => {
    await mockRoutes(page, ACCOUNT_ACTIVE_CLEAN)
    await gotoPage(page)

    await expect(
      page.getByRole("heading", { name: /Compte entièrement actif/i }),
    ).toBeVisible({ timeout: 10_000 })

    // Both capabilities active
    await expect(page.getByText("Paiements entrants", { exact: true })).toBeVisible()
    await expect(page.getByText("Virements sortants", { exact: true })).toBeVisible()
    // Two green "Actif" badges (one per capability)
    const actifBadges = page.getByText("Actif", { exact: true })
    await expect(actifBadges).toHaveCount(2)
  })

  test("currently_due populated → banner shows count + orange 'Vérification en cours'", async ({
    page,
  }) => {
    await mockRoutes(page, {
      ...ACCOUNT_ACTIVE_CLEAN,
      requirements_currently_due: ["individual.verification.document"],
      requirements_count: 1,
    })
    await gotoPage(page)

    await expect(
      page.getByRole("heading", { name: /Vérification en cours/i }),
    ).toBeVisible({ timeout: 10_000 })

    await expect(page.getByText("1 information à compléter")).toBeVisible()
  })

  test("currently_due multiple → pluralizes 'X informations à compléter'", async ({
    page,
  }) => {
    await mockRoutes(page, {
      ...ACCOUNT_ACTIVE_CLEAN,
      requirements_currently_due: [
        "individual.verification.document",
        "individual.phone",
        "external_account",
      ],
      requirements_count: 3,
    })
    await gotoPage(page)

    await expect(
      page.getByText("3 informations à compléter"),
    ).toBeVisible({ timeout: 10_000 })
  })

  test("past_due populated → shows red 'Action urgente requise'", async ({ page }) => {
    await mockRoutes(page, {
      ...ACCOUNT_ACTIVE_CLEAN,
      requirements_past_due: ["individual.verification.document"],
      requirements_count: 1,
    })
    await gotoPage(page)

    await expect(
      page.getByRole("heading", { name: /Action urgente requise/i }),
    ).toBeVisible({ timeout: 10_000 })
  })

  test("charges_enabled=false → 'Paiements entrants' shows 'En attente'", async ({
    page,
  }) => {
    await mockRoutes(page, {
      ...ACCOUNT_ACTIVE_CLEAN,
      charges_enabled: false,
      payouts_enabled: true,
    })
    await gotoPage(page)

    await expect(page.getByText("Paiements entrants")).toBeVisible({ timeout: 10_000 })
    // At least one "En attente" badge visible for charges
    const enAttenteBadges = page.getByText("En attente", { exact: true })
    await expect(enAttenteBadges.first()).toBeVisible()
  })

  test("both capabilities disabled → both show 'En attente'", async ({ page }) => {
    await mockRoutes(page, {
      ...ACCOUNT_ACTIVE_CLEAN,
      charges_enabled: false,
      payouts_enabled: false,
    })
    await gotoPage(page)

    const enAttenteBadges = page.getByText("En attente", { exact: true })
    await expect(enAttenteBadges).toHaveCount(2)
  })

  test("details_submitted=false → onboarding mode, Gérer mes informations NOT shown", async ({
    page,
  }) => {
    await mockRoutes(page, {
      ...ACCOUNT_ACTIVE_CLEAN,
      details_submitted: false,
    })
    await gotoPage(page)

    // Onboarding section header visible
    await expect(page.getByText(/Finaliser la vérification/i)).toBeVisible({
      timeout: 10_000,
    })

    // Management section (dashboard) is NOT visible
    await expect(page.getByText(/Gérer mes informations/i)).not.toBeVisible()
  })

  test("details_submitted=true → dashboard mode, Gérer mes informations visible", async ({
    page,
  }) => {
    await mockRoutes(page, ACCOUNT_ACTIVE_CLEAN)
    await gotoPage(page)

    await expect(
      page.getByRole("heading", { name: /Gérer mes informations/i }),
    ).toBeVisible({ timeout: 10_000 })

    // Onboarding header NOT visible (we're in dashboard mode)
    await expect(page.getByText(/Finaliser la vérification/i)).not.toBeVisible()
  })

  test("account_id displayed in status card (desktop viewport)", async ({ page }) => {
    await page.setViewportSize({ width: 1280, height: 800 })
    await mockRoutes(page, {
      ...ACCOUNT_ACTIVE_CLEAN,
      account_id: "acct_1TIsgNPyy7y81FsB",
    })
    await gotoPage(page)

    await expect(page.getByText("acct_1TIsgNPyy7y81FsB")).toBeVisible({
      timeout: 10_000,
    })
  })

  test("only eventually_due → still shows 'Vérification en cours' with count", async ({
    page,
  }) => {
    await mockRoutes(page, {
      ...ACCOUNT_ACTIVE_CLEAN,
      requirements_eventually_due: ["individual.verification.additional_document"],
      requirements_count: 1,
    })
    await gotoPage(page)

    await expect(
      page.getByRole("heading", { name: /Vérification en cours/i }),
    ).toBeVisible({ timeout: 10_000 })
    await expect(page.getByText("1 information à compléter")).toBeVisible()
  })

  test("polling refreshes status every 10s", async ({ page }) => {
    let callCount = 0
    await page.route(/\/api\/v1\/payment-info\/account-status/, async (route: Route) => {
      callCount++
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(ACCOUNT_ACTIVE_CLEAN),
      })
    })
    await page.route(/\/api\/v1\/auth\/me/, async (route: Route) => {
      await route.fulfill({ status: 401, body: "{}" })
    })

    await gotoPage(page)
    await expect(
      page.getByRole("heading", { name: /Compte entièrement actif/i }),
    ).toBeVisible({ timeout: 10_000 })

    // Initial call should have happened
    expect(callCount).toBeGreaterThanOrEqual(1)

    // Wait for polling interval (10s) + margin
    await page.waitForTimeout(11_500)

    // At least 2 calls now (initial + 1 polling cycle)
    expect(callCount).toBeGreaterThanOrEqual(2)
  })
})
