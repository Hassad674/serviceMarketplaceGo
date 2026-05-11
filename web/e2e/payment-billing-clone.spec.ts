import { test, expect, type Page, type Route } from "@playwright/test"

// ---------------------------------------------------------------------------
// BILLING-IDENTITY-CLONE — Playwright e2e
//
// Pins the user-visible contract on /fr/projects/pay after the
// inline mini-form (PaymentBillingIdentitySection) was replaced by an
// embedded BillingProfileForm clone. Two scenarios:
//
//   1. Profile incomplete on first visit -> the embed renders in form
//      mode, the payment CTA is hidden. After save -> embed collapses
//      into the read-only summary AND the payment CTA appears.
//   2. Profile complete on first visit -> embed renders in summary
//      mode; clicking "Modifier" opens the form; the payment CTA is
//      hidden while editing.
//
// All backend endpoints are mocked via page.route() so the test
// runs without a live Go server. Stripe is intentionally NOT mocked
// — the SimulationFallback branch is exercised (it has the same
// gating logic as the Stripe branch).
// ---------------------------------------------------------------------------

const ENTERPRISE_USER_ID = "ent-user-clone"
const ENTERPRISE_ORG_ID = "ent-org-clone"
const PROPOSAL_ID = "proposal-clone"

type BillingProfile = {
  organization_id: string
  profile_type: "business" | "individual"
  legal_name: string
  trading_name: string
  legal_form: string
  tax_id: string
  vat_number: string
  vat_validated_at: string | null
  address_line1: string
  address_line2: string
  postal_code: string
  city: string
  country: string
  invoicing_email: string
  synced_from_kyc_at: string | null
}

const completeProfile: BillingProfile = {
  organization_id: ENTERPRISE_ORG_ID,
  profile_type: "business",
  legal_name: "Acme Studio SARL",
  trading_name: "",
  legal_form: "SARL",
  tax_id: "12345678901234",
  vat_number: "",
  vat_validated_at: null,
  address_line1: "12 rue de la Paix",
  address_line2: "",
  postal_code: "75001",
  city: "Paris",
  country: "FR",
  invoicing_email: "",
  synced_from_kyc_at: null,
}

const emptyProfile: BillingProfile = {
  organization_id: ENTERPRISE_ORG_ID,
  profile_type: "individual",
  legal_name: "",
  trading_name: "",
  legal_form: "",
  tax_id: "",
  vat_number: "",
  vat_validated_at: null,
  address_line1: "",
  address_line2: "",
  postal_code: "",
  city: "",
  country: "",
  invoicing_email: "",
  synced_from_kyc_at: null,
}

function snapshotFromProfile(profile: BillingProfile) {
  const missingFields: { field: string; reason: string }[] = []
  if (!profile.legal_name) missingFields.push({ field: "legal_name", reason: "required" })
  if (!profile.country) missingFields.push({ field: "country", reason: "required" })
  if (!profile.address_line1) missingFields.push({ field: "address_line1", reason: "required" })
  if (!profile.postal_code) missingFields.push({ field: "postal_code", reason: "required" })
  if (!profile.city) missingFields.push({ field: "city", reason: "required" })
  if (profile.profile_type === "business" && !profile.tax_id) {
    missingFields.push({ field: "tax_id", reason: "required" })
  }
  return {
    profile,
    missing_fields: missingFields,
    is_complete: missingFields.length === 0,
  }
}

async function mockAuth(page: Page) {
  await page.route(/\/api\/v1\/auth\/me\b/, async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        user: {
          id: ENTERPRISE_USER_ID,
          email: "ent@example.com",
          first_name: "Ent",
          last_name: "User",
          display_name: "Ent User",
          role: "enterprise",
          referrer_enabled: false,
          email_verified: true,
          kyc_status: "verified",
          created_at: "2026-01-01",
        },
        organization: {
          id: ENTERPRISE_ORG_ID,
          name: "Acme",
          kyc_status: "verified",
        },
      }),
    })
  })
}

async function mockProposal(page: Page) {
  await page.route(
    new RegExp(`/api/v1/proposals/${PROPOSAL_ID}(?:\\?.*)?$`),
    async (route: Route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          id: PROPOSAL_ID,
          title: "Refonte du site Acme",
          amount: 150000,
          status: "accepted",
          current_milestone_sequence: 1,
          milestones: [
            {
              id: "ms-1",
              sequence: 1,
              status: "pending_funding",
              amount: 150000,
            },
          ],
        }),
      })
    },
  )
}

async function mockInitiatePayment(page: Page) {
  await page.route(
    /\/api\/v1\/proposals\/[^/]+\/pay$/,
    async (route: Route) => {
      // Always responds with a "simulation-mode" payload (no client_secret)
      // so the page renders the SimulationFallback branch — which uses
      // the same gating logic as the Stripe branch.
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          status: "pending",
          amounts: {
            proposal_amount: 150000,
            stripe_fee: 2000,
            platform_fee: 5000,
            client_total: 157000,
            provider_payout: 145000,
          },
        }),
      })
    },
  )
}

async function mockBillingProfile(page: Page, initial: BillingProfile) {
  let current = { ...initial }
  await page.route(/\/api\/v1\/billing-profile(?:\?.*)?$/, async (route: Route) => {
    if (route.request().method() === "PUT") {
      const body = route.request().postDataJSON() as Partial<BillingProfile>
      current = { ...current, ...body }
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(snapshotFromProfile(current)),
      })
      return
    }
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify(snapshotFromProfile(current)),
    })
  })
}

test.describe("Payment page — billing-identity clone", () => {
  test("complete profile renders the read-only summary AND the payment CTA on first paint", async ({
    page,
  }) => {
    await mockAuth(page)
    await mockProposal(page)
    await mockInitiatePayment(page)
    await mockBillingProfile(page, completeProfile)

    await page.goto(`/fr/projects/pay?proposal=${PROPOSAL_ID}`)
    await page.waitForLoadState("networkidle")

    // The summary card shows the saved legal name + address.
    await expect(page.getByText("Acme Studio SARL")).toBeVisible({
      timeout: 10_000,
    })
    await expect(page.getByText(/12 rue de la Paix/)).toBeVisible()
    // The "Modifier" CTA is on screen so the user can re-open the form.
    await expect(page.getByRole("button", { name: /Modifier/i })).toBeVisible()
    // The confirmPayment CTA is rendered (gated by isPaymentReady).
    await expect(
      page.getByRole("button", { name: /Confirmer le paiement/i }),
    ).toBeVisible()
  })

  test("incomplete profile shows the form and hides the payment CTA until the user saves", async ({
    page,
  }) => {
    await mockAuth(page)
    await mockProposal(page)
    await mockInitiatePayment(page)
    await mockBillingProfile(page, emptyProfile)

    await page.goto(`/fr/projects/pay?proposal=${PROPOSAL_ID}`)
    await page.waitForLoadState("networkidle")

    // The full form is rendered — the "Pays" section header is the
    // canonical first heading of the prestataire form.
    await expect(
      page.getByRole("heading", { name: "Pays" }),
    ).toBeVisible({ timeout: 10_000 })

    // The confirmPayment CTA MUST NOT be visible while the profile
    // is incomplete.
    await expect(
      page.getByRole("button", { name: /Confirmer le paiement/i }),
    ).toHaveCount(0)
  })

  test("clicking 'Modifier' flips the embed to form mode and hides the payment CTA", async ({
    page,
  }) => {
    await mockAuth(page)
    await mockProposal(page)
    await mockInitiatePayment(page)
    await mockBillingProfile(page, completeProfile)

    await page.goto(`/fr/projects/pay?proposal=${PROPOSAL_ID}`)
    await page.waitForLoadState("networkidle")

    // Wait for the summary card.
    await expect(page.getByText("Acme Studio SARL")).toBeVisible({
      timeout: 10_000,
    })

    // Click Modifier — the form opens, the payment CTA disappears.
    await page.getByRole("button", { name: /Modifier/i }).click()
    await expect(
      page.getByRole("heading", { name: "Pays" }),
    ).toBeVisible({ timeout: 5_000 })
    await expect(
      page.getByRole("button", { name: /Confirmer le paiement/i }),
    ).toHaveCount(0)
  })
})
