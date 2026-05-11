import { expect, test } from "@playwright/test"

import { clearAuth, registerProvider } from "./helpers/auth"

// ---------------------------------------------------------------------------
// Wallet — D1+D2 commission Retirer flow
//
// Verifies the apporteur-side "Mes commissions" section: a commission
// stuck in pending_kyc renders a Retirer button; clicking the button
// opens the KYC modal (because the test provider has no Stripe Connect
// account, the backend returns 422 kyc_required); closing the modal
// leaves the row unchanged.
//
// The test mocks the GET /api/v1/wallet response so we have a
// deterministic commission row to interact with — seeding a real
// commission would require a full propose/accept/approve flow which is
// covered in referral-commission-flow.spec.ts.
// ---------------------------------------------------------------------------

test.describe("Wallet — D1+D2 commission Retirer fallback", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/")
    await clearAuth(page)
  })

  test("apporteur sees a Retirer button on a pending_kyc commission and the click opens the KYC modal", async ({
    page,
  }) => {
    await registerProvider(page)

    // Mock GET /wallet to inject a deterministic commission row.
    await page.route("**/api/v1/wallet", async (route) => {
      if (route.request().method() !== "GET") {
        await route.fallback()
        return
      }
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          stripe_account_id: "",
          charges_enabled: false,
          payouts_enabled: false,
          escrow_amount: 0,
          available_amount: 0,
          transferred_amount: 0,
          records: [],
          commissions: {
            pending_cents: 0,
            pending_kyc_cents: 5000,
            paid_cents: 0,
            clawed_back_cents: 0,
            currency: "EUR",
          },
          commission_records: [
            {
              id: "00000000-0000-0000-0000-000000000001",
              referral_id: "",
              proposal_id: "",
              milestone_id: "",
              gross_amount_cents: 100000,
              commission_cents: 5000,
              currency: "EUR",
              status: "pending_kyc",
              created_at: "2026-05-01T10:00:00Z",
              retire_eligible: true,
            },
          ],
        }),
      })
    })

    // Mock the retry endpoint to return 422 kyc_required — simulating
    // the production case where the apporteur still has not finished
    // onboarding.
    await page.route(
      "**/api/v1/wallet/commissions/*/retry",
      async (route) => {
        await route.fulfill({
          status: 422,
          contentType: "application/json",
          body: JSON.stringify({
            error: {
              code: "kyc_required",
              message:
                "Termine d'abord ton onboarding Stripe pour pouvoir recevoir cette commission.",
            },
            onboarding_url: "https://stripe.com/connect/onboarding/abc",
            redirect: "/payment-info",
          }),
        })
      },
    )

    await page.goto("/wallet")
    await page.waitForLoadState("networkidle")

    // The commission section must render the Retirer button.
    const retireBtn = page.getByRole("button", {
      name: /Retirer cette commission/i,
    })
    await expect(retireBtn).toBeVisible({ timeout: 15000 })

    await retireBtn.click()

    // The KYC modal must open with the title from
    // commission-kyc-required-modal.tsx.
    await expect(
      page.getByRole("heading", {
        name: /Termine ton KYC pour recevoir ta commission/i,
      }),
    ).toBeVisible({ timeout: 5000 })

    // The deep-link CTA carries the onboarding URL the backend
    // returned (asserted by href so we don't navigate during the
    // test).
    const cta = page.getByRole("link", { name: /Terminer mon KYC/i })
    await expect(cta).toHaveAttribute(
      "href",
      "https://stripe.com/connect/onboarding/abc",
    )
    await expect(cta).toHaveAttribute("target", "_blank")

    // Close the modal — the commission row stays in place, the
    // section's headings are still visible.
    await page.getByRole("button", { name: /Plus tard/i }).click()
    await expect(
      page.getByRole("heading", {
        name: /Termine ton KYC pour recevoir ta commission/i,
      }),
    ).not.toBeVisible()
    await expect(retireBtn).toBeVisible()
  })
})
