import { test, expect } from "@playwright/test"

// E2E coverage for the invoicing surfaces (Phase 7).
//
// This spec is intentionally light on backend assumptions. Phase 7 ships
// only the operator-side UI; backend fixtures for invoice rows, billing
// profile presets, and Stripe KYC sync are still owned by the test
// harness work in Phase 10. Until those fixtures land, the only checks
// that can run reliably in CI are the structural ones (routes resolve,
// pages render their landmarks). The richer interaction tests are
// guarded behind `INVOICING_E2E_FIXTURES=1` so a future seed can
// flip the switch without rewriting this file.

const FIXTURES_AVAILABLE = process.env.INVOICING_E2E_FIXTURES === "1"

test.describe("invoicing — structural", () => {
  test("billing profile page is reachable", async ({ page }) => {
    await page.goto("/settings/billing-profile")
    // The page either renders the form (authenticated) or bounces to
    // login. Both outcomes prove the route exists. We accept either
    // landmark to keep the spec stable across auth states in CI.
    const heading = page.getByRole("heading", {
      name: /Profil de facturation|Connexion|Login/i,
    })
    await expect(heading).toBeVisible()
  })

  test("invoices page is reachable", async ({ page }) => {
    await page.goto("/invoices")
    const heading = page.getByRole("heading", {
      name: /Mes factures|Connexion|Login/i,
    })
    await expect(heading).toBeVisible()
  })
})

test.describe("invoicing — fixture-gated flows", () => {
  test.skip(
    !FIXTURES_AVAILABLE,
    "Set INVOICING_E2E_FIXTURES=1 with a seeded provider account to run.",
  )

  test("withdraw is gated until the billing profile is complete", async ({
    page,
  }) => {
    // 1. Provider with empty billing profile lands on /wallet.
    await page.goto("/wallet")
    await page.getByRole("button", { name: /Retirer/i }).click()
    await expect(
      page.getByRole("heading", {
        name: /Complète ton profil de facturation/i,
      }),
    ).toBeVisible()
    // 2. Click the CTA — we end up on the billing profile page.
    await page.getByRole("button", { name: /Compléter mon profil/i }).click()
    await expect(page).toHaveURL(/\/settings\/billing-profile/)
    // 3. Pre-fill from Stripe and save.
    await page.getByRole("button", { name: /Pré-remplir depuis Stripe/i }).click()
    await page.getByRole("button", { name: /Enregistrer/i }).click()
    await expect(page.getByText(/Profil enregistré/i)).toBeVisible()
    // 4. Back to wallet — withdraw is no longer gated.
    await page.goto("/wallet")
    await page.getByRole("button", { name: /Retirer/i }).click()
    await expect(
      page.getByRole("heading", {
        name: /Complète ton profil de facturation/i,
      }),
    ).not.toBeVisible({ timeout: 1_000 })
  })

  test("invoice list renders and exposes a download link", async ({ page }) => {
    await page.goto("/invoices")
    await expect(page.getByRole("heading", { name: /Mes factures/i })).toBeVisible()
    const link = page.getByRole("link", { name: /Télécharger/i }).first()
    await expect(link).toBeVisible()
    const href = await link.getAttribute("href")
    expect(href).toMatch(/\/api\/v1\/me\/invoices\/[^/]+\/pdf/)
  })
})
