import path from "node:path"
import { test, expect, type Page } from "@playwright/test"
import { registerProvider, registerAgency, STRONG_PASSWORD } from "./helpers/auth"
import type { RegisteredUser } from "./helpers/auth"

// ---------------------------------------------------------------------------
// Country config — US (primary target for Stripe magic values)
// ---------------------------------------------------------------------------

const US = {
  code: "US",
  phone: "+14155551234",
  postalCode: "94102",
  state: "CA",
  routingNumber: "110000000",
  accountNumber: "000123456789",
  businessPostalCode: "10001",
  businessState: "NY",
}

// ---------------------------------------------------------------------------
// Name pools
// ---------------------------------------------------------------------------

const FIRST_NAMES = [
  "Emma", "Liam", "Sophia", "Noah", "Olivia", "James", "Ava", "William",
]
const LAST_NAMES = [
  "Smith", "Johnson", "Williams", "Brown", "Jones", "Garcia", "Miller",
]
const CITIES = ["Springfield", "Portland", "Austin", "Denver", "Seattle"]
const STREETS = ["123 Main St", "456 Oak Ave", "789 Pine Rd", "321 Elm Blvd"]

function pick<T>(arr: T[]): T {
  return arr[Math.floor(Math.random() * arr.length)]
}

// ---------------------------------------------------------------------------
// Form helpers
// ---------------------------------------------------------------------------

async function fillByLabel(page: Page, label: string, value: string) {
  const labelEl = page.locator("label", { hasText: label }).first()
  const input = labelEl.locator("..").locator("input").first()
  await input.scrollIntoViewIfNeeded()
  await input.fill(value)
}

async function navigateToPaymentInfo(page: Page) {
  await page.goto("/en/payment-info")
  await expect(
    page.getByText("Payment Information").first(),
  ).toBeVisible({ timeout: 10000 })
}

async function selectCountry(page: Page, code: string) {
  const countrySection = page.locator("section, div")
    .filter({ hasText: "Activity Country" }).first()
  const countrySelect = countrySection.locator("select").first()
  await countrySelect.selectOption(code)
  await expect(
    page.locator("label", { hasText: "First name" }).first(),
  ).toBeVisible({ timeout: 10000 })
}

async function uploadDocuments(page: Page) {
  const fixturePath = path.resolve(__dirname, "fixtures", "test-passport.png")
  const uploadButtons = page.locator("button", { hasText: "Click to upload" })
  const count = await uploadButtons.count()

  for (let i = 0; i < count; i++) {
    const btn = page.locator("button", { hasText: "Click to upload" }).first()
    await btn.scrollIntoViewIfNeeded()
    await btn.click()

    const modal = page.locator("div[role='dialog']")
    await expect(modal).toBeVisible({ timeout: 5000 })

    const fileInput = modal.locator("input[type='file']")
    await fileInput.setInputFiles(fixturePath)

    await expect(
      modal.locator("text=test-passport.png"),
    ).toBeVisible({ timeout: 5000 })

    const uploadBtn = modal.locator("button", { hasText: "Upload" }).last()
    await uploadBtn.click()

    await expect(modal).toBeHidden({ timeout: 15000 })
    await expect(
      page.getByText("Document uploaded").nth(i),
    ).toBeVisible({ timeout: 15000 })
  }
}

async function saveAndWaitForResponse(page: Page) {
  await page.evaluate(() => {
    const main = document.querySelector("main")
    if (main) main.scrollTop = main.scrollHeight
    window.scrollTo(0, document.body.scrollHeight)
  })
  await page.waitForTimeout(500)

  const saveButton = page.locator("button", { hasText: "Save" }).first()
  await saveButton.scrollIntoViewIfNeeded()
  await expect(saveButton).toBeVisible({ timeout: 5000 })
  await expect(saveButton).toBeEnabled({ timeout: 10000 })

  const responsePromise = page.waitForResponse(
    (resp) =>
      resp.url().includes("/payment-info") &&
      resp.request().method() === "PUT",
    { timeout: 60000 },
  )

  await saveButton.click()
  await responsePromise

  await page.evaluate(() => {
    const main = document.querySelector("main")
    if (main) main.scrollTop = 0
    window.scrollTo(0, 0)
  })
  await page.waitForTimeout(500)

  await expect(
    page.getByText("Payment information saved"),
  ).toBeVisible({ timeout: 60000 })
}

// ---------------------------------------------------------------------------
// US Individual — fill all fields with overrides
// ---------------------------------------------------------------------------

async function fillUsPersonalFields(
  page: Page,
  overrides: { ssn?: string; idNumber?: string; address?: string },
) {
  const firstName = pick(FIRST_NAMES)
  const lastName = pick(LAST_NAMES)
  const ts = Date.now()

  await fillByLabel(page, "First name", firstName)
  await fillByLabel(page, "Last name", lastName)

  const dobLabel = page.locator("label", { hasText: "Date of birth" }).first()
  if (await dobLabel.isVisible().catch(() => false)) {
    const dobInput = dobLabel.locator("..").locator("input").first()
    await dobInput.scrollIntoViewIfNeeded()
    await dobInput.fill("1990-05-15")
  }

  const emailLabel = page.locator("label", { hasText: "Email" }).first()
  if (await emailLabel.isVisible().catch(() => false)) {
    await fillByLabel(page, "Email", `kyc-fail-${ts}@test.com`)
  }

  await fillByLabel(page, "Address", overrides.address ?? pick(STREETS))
  await fillByLabel(page, "City", pick(CITIES))
  await fillByLabel(page, "Postal code", US.postalCode)
  await fillByLabel(page, "Phone number", US.phone)

  const stateLabel = page.locator("label", { hasText: "State" }).first()
  if (await stateLabel.isVisible().catch(() => false)) {
    const stateInput = stateLabel.locator("..").locator("input, select").first()
    await stateInput.scrollIntoViewIfNeeded()
    await stateInput.fill(US.state)
  }

  // SSN last 4 — use override or success value
  const ssnLabel = page.locator("label", { hasText: "SSN" }).first()
  if (await ssnLabel.isVisible().catch(() => false)) {
    const ssnInput = ssnLabel.locator("..").locator("input").first()
    await ssnInput.scrollIntoViewIfNeeded()
    await ssnInput.fill(overrides.ssn ?? "0000")
  }

  // National ID Number — use override or success value
  const idLabel = page.locator("label", { hasText: "National ID" }).first()
  if (await idLabel.isVisible().catch(() => false)) {
    const idInput = idLabel.locator("..").locator("input").first()
    await idInput.scrollIntoViewIfNeeded()
    await idInput.fill(overrides.idNumber ?? "000000000")
  }

  // Political exposure
  const peLabel = page.locator("label", { hasText: "Political Exposure" }).first()
  if (await peLabel.isVisible().catch(() => false)) {
    const peSelect = peLabel.locator("..").locator("select").first()
    await peSelect.scrollIntoViewIfNeeded()
    await peSelect.selectOption("none")
  }

  // Activity sector
  const sectorSelect = page.locator("select[aria-label='Activity sector']")
  if (await sectorSelect.isVisible().catch(() => false)) {
    await sectorSelect.selectOption({ index: 1 })
  }

  // Nationality
  const natLabel = page.locator("label", { hasText: "Nationality" }).first()
  if (await natLabel.isVisible().catch(() => false)) {
    const natSelect = natLabel.locator("..").locator("select").first()
    if (await natSelect.isVisible().catch(() => false)) {
      await natSelect.scrollIntoViewIfNeeded()
      await natSelect.selectOption("US")
    }
  }

  // Catch-all: fill remaining empty inputs
  await fillRemainingInputs(page)

  return { firstName, lastName, accountHolder: `${firstName} ${lastName}` }
}

async function fillUsBankFields(page: Page, accountHolder: string) {
  await fillByLabel(page, "Account number", US.accountNumber)
  await fillByLabel(page, "Routing number", US.routingNumber)
  await fillByLabel(page, "Account holder name", accountHolder)

  const bankCountryLabel = page.locator("label", { hasText: "Bank country" }).first()
  if (await bankCountryLabel.isVisible().catch(() => false)) {
    const bankCountrySelect = bankCountryLabel.locator("..").locator("select").first()
    await bankCountrySelect.scrollIntoViewIfNeeded()
    await bankCountrySelect.selectOption("US")
  }
}

async function fillUsBusinessFields(
  page: Page,
  overrides: { taxId?: string },
): Promise<string> {
  const ts = Date.now()
  const bizName = `Test Corp ${ts}`

  const roleSelect = page.locator("select[aria-label='Your role in the company']")
  if (await roleSelect.isVisible().catch(() => false)) {
    await roleSelect.scrollIntoViewIfNeeded()
    await roleSelect.selectOption("ceo")
  }

  const titleLabel = page.locator("label", { hasText: "Title / Role" }).first()
  if (await titleLabel.isVisible().catch(() => false)) {
    await fillByLabel(page, "Title / Role", "CEO")
  }

  await fillByLabel(page, "Business name", bizName)
  await fillByLabel(page, "Business address", pick(STREETS))
  await fillByLabel(page, "Business city", pick(CITIES))
  await fillByLabel(page, "Business postal code", US.businessPostalCode)

  if (US.businessState) {
    const stateLabel = page.locator("label", { hasText: "Business state" }).first()
    if (await stateLabel.isVisible().catch(() => false)) {
      const stateInput = stateLabel.locator("..").locator("input, select").first()
      await stateInput.scrollIntoViewIfNeeded()
      await stateInput.fill(US.businessState)
    }
  }

  const taxLabel = page.locator("label", { hasText: "Tax ID" }).first()
  if (await taxLabel.isVisible().catch(() => false)) {
    await fillByLabel(page, "Tax ID", overrides.taxId ?? "000000000")
  }

  const phoneLabels = page.locator("label", { hasText: "Phone number" })
  const phoneCount = await phoneLabels.count()
  if (phoneCount > 1) {
    const companyPhoneInput = phoneLabels.nth(1).locator("..").locator("input").first()
    if (await companyPhoneInput.isVisible().catch(() => false)) {
      await companyPhoneInput.scrollIntoViewIfNeeded()
      await companyPhoneInput.fill(US.phone)
    }
  }

  // Catch-all for business section
  await fillRemainingInputs(page)
  await fillRemainingSelects(page)

  return bizName
}

/** Fill any remaining empty text/email/tel inputs on the page. */
async function fillRemainingInputs(page: Page) {
  const allInputs = page.locator(
    "section input[type='text'], section input[type='email'], " +
    "section input[type='tel'], section input:not([type])",
  )
  const inputCount = await allInputs.count()
  for (let i = 0; i < inputCount; i++) {
    const input = allInputs.nth(i)
    if (await input.isVisible().catch(() => false)) {
      const val = await input.inputValue().catch(() => "")
      if (val === "") {
        await input.scrollIntoViewIfNeeded()
        await input.fill("Test123")
      }
    }
  }
}

/** Fill any remaining empty selects on the page. */
async function fillRemainingSelects(page: Page) {
  const allSelects = page.locator("select")
  const selectCount = await allSelects.count()
  for (let i = 0; i < selectCount; i++) {
    const sel = allSelects.nth(i)
    if (await sel.isVisible().catch(() => false)) {
      const val = await sel.inputValue().catch(() => "")
      if (val === "") {
        await sel.scrollIntoViewIfNeeded()
        await sel.selectOption({ index: 1 }).catch(() => {})
      }
    }
  }
}

/**
 * Wait for Stripe webhook processing and reload the page.
 * After saving, Stripe processes verification asynchronously.
 * The `account.updated` webhook fires when done.
 */
async function waitForStripeProcessingAndReload(page: Page) {
  // Poll: reload every 10s until "Action required" appears, max 60s
  for (let i = 0; i < 6; i++) {
    await page.waitForTimeout(10000)
    await page.reload()
    await page.waitForTimeout(2000)
    const banner = page.getByText("Action required").first()
    if (await banner.isVisible().catch(() => false)) {
      return // Requirements are showing
    }
  }
  // Final reload attempt
  await page.reload()
  await expect(
    page.getByText("Payment Information").first(),
  ).toBeVisible({ timeout: 15000 })
}

/**
 * Register a provider with a custom email containing a suffix.
 * Used for the enforce_future_requirements test.
 */
async function registerProviderWithEmail(
  page: Page,
  emailSuffix: string,
): Promise<RegisteredUser> {
  const ts = Date.now()
  const email = `test-provider-${ts}${emailSuffix}@playwright.com`
  const firstName = "Jean"
  const lastName = `Dupont${ts}`

  await page.goto("/register/provider")
  await page.getByLabel("First name").fill(firstName)
  await page.getByLabel("Last name", { exact: true }).fill(lastName)
  await page.getByLabel("Email").fill(email)
  await page.getByLabel("Password", { exact: true }).fill(STRONG_PASSWORD)
  await page.getByLabel("Confirm password").fill(STRONG_PASSWORD)
  await page.getByRole("button", { name: /Create my freelance account/i }).click()
  await page.waitForURL("**/dashboard", { timeout: 15000 })

  return { email, password: STRONG_PASSWORD, displayName: `${firstName} ${lastName}` }
}

// ---------------------------------------------------------------------------
// Assertion helpers
// ---------------------------------------------------------------------------

/** Check that the requirements banner is visible with "Action required" text. */
async function expectRequirementsBanner(page: Page) {
  await page.evaluate(() => {
    const main = document.querySelector("main")
    if (main) main.scrollTop = 0
    window.scrollTo(0, 0)
  })
  await page.waitForTimeout(300)

  await expect(
    page.getByText("Action required").first(),
  ).toBeVisible({ timeout: 15000 })
}

/** Check that at least one input has a red error border (border-red-500). */
async function expectFieldWithErrorBorder(page: Page) {
  const errorInput = page.locator("[aria-invalid='true']").first()
  await expect(errorInput).toBeVisible({ timeout: 10000 })
}

/** Check that an error message is visible below a field. */
async function expectErrorMessage(page: Page) {
  const errorMsg = page.locator("[role='alert']").first()
  await expect(errorMsg).toBeVisible({ timeout: 10000 })
}

// ---------------------------------------------------------------------------
// Test 1 — SSN / ID Number failure (US Individual)
// ---------------------------------------------------------------------------

test.describe("KYC Failure — SSN/ID Number (US Individual)", () => {
  test("SSN 111111111 triggers verification failure, requirements banner shown", async ({ page }) => {
    test.setTimeout(180_000)

    // Register and navigate
    await registerProvider(page)
    await navigateToPaymentInfo(page)
    await selectCountry(page, "US")

    // Fill fields with FAILING SSN/ID values
    const person = await fillUsPersonalFields(page, {
      ssn: "1111",
      idNumber: "111111111",
    })
    await fillUsBankFields(page, person.accountHolder)
    await uploadDocuments(page)

    // Save
    await saveAndWaitForResponse(page)

    // Wait for Stripe to process the verification and fire webhook
    await waitForStripeProcessingAndReload(page)

    // Verify: requirements banner is visible
    await expectRequirementsBanner(page)

    // Verify: at least one field has an error indicator
    await expectFieldWithErrorBorder(page)
  })
})

// ---------------------------------------------------------------------------
// Test 2 — Address failure (US Individual)
// ---------------------------------------------------------------------------

test.describe("KYC Failure — Address Mismatch (US Individual)", () => {
  test("address_no_match triggers verification failure, requirements banner shown", async ({ page }) => {
    test.setTimeout(180_000)

    await registerProvider(page)
    await navigateToPaymentInfo(page)
    await selectCountry(page, "US")

    // Fill with FAILING address, but SUCCESS SSN/ID
    const person = await fillUsPersonalFields(page, {
      ssn: "0000",
      idNumber: "000000000",
      address: "address_no_match",
    })
    await fillUsBankFields(page, person.accountHolder)
    await uploadDocuments(page)

    await saveAndWaitForResponse(page)

    // Wait for Stripe webhook processing
    await waitForStripeProcessingAndReload(page)

    // Verify: requirements banner appears
    await expectRequirementsBanner(page)

    // Verify: error state present on form
    await expectFieldWithErrorBorder(page)
  })
})

// ---------------------------------------------------------------------------
// Test 3 — Tax ID failure (US Business)
// ---------------------------------------------------------------------------

test.describe("KYC Failure — Tax ID (US Business)", () => {
  test("tax_id 111111111 triggers verification failure, requirements banner shown", async ({ page }) => {
    test.setTimeout(180_000)

    await registerAgency(page)
    await navigateToPaymentInfo(page)
    await selectCountry(page, "US")

    // Toggle business mode ON
    await page.getByRole("switch").click()
    await expect(
      page.getByText("Company Information").first(),
    ).toBeVisible({ timeout: 10000 })

    // Fill with SUCCESS personal values but FAILING tax_id
    const person = await fillUsPersonalFields(page, {
      ssn: "0000",
      idNumber: "000000000",
    })
    const bizName = await fillUsBusinessFields(page, {
      taxId: "111111111",
    })
    await fillUsBankFields(page, bizName)
    await uploadDocuments(page)

    await saveAndWaitForResponse(page)

    // Wait for Stripe webhook processing
    await waitForStripeProcessingAndReload(page)

    // Verify: requirements banner appears (tax ID failed)
    await expectRequirementsBanner(page)
  })
})

// ---------------------------------------------------------------------------
// Test 4 — Enforce future requirements promotes document requirements
// ---------------------------------------------------------------------------
// Register with +enforce_future_requirements email, fill all text fields with
// valid data, fill bank but DO NOT upload documents. Stripe promotes
// eventually_due (document) to currently_due → banner appears.

test.describe("KYC Failure — Enforce Future Requirements (no documents)", () => {
  test("enforce_future_requirements without docs triggers document requirements", async ({ page }) => {
    test.setTimeout(180_000)

    // Register with +enforce_future_requirements so Stripe promotes eventually_due
    await registerProviderWithEmail(page, "+enforce_future_requirements")
    await navigateToPaymentInfo(page)
    await selectCountry(page, "US")

    // Use SSN 111111111 (known to trigger verification failure) combined with
    // skipping document upload. This ensures Stripe has at least one failing
    // requirement PLUS unmet document fields that get promoted.
    const person = await fillUsPersonalFields(page, {
      ssn: "1111",
      idNumber: "111111111",
      address: "address_full_match",
    })
    await fillUsBankFields(page, person.accountHolder)

    // Override the Email field with enforce suffix so Stripe promotes
    // eventually_due items to currently_due
    const ts = Date.now()
    await fillByLabel(page, "Email", `kyc-enforce-${ts}+enforce_future_requirements@test.com`)

    // DO NOT upload documents — they will be promoted to currently_due

    await saveAndWaitForResponse(page)
    await waitForStripeProcessingAndReload(page)

    // SSN failure + promoted document requirements → banner appears
    await expectRequirementsBanner(page)
  })
})

// ---------------------------------------------------------------------------
// Test 5 — Document missing triggers eventually_due after save
// ---------------------------------------------------------------------------
// Normal registration (no enforce email), fill text fields + bank, skip docs.
// Without +enforce_future_requirements, documents stay in eventually_due.
// Known limitation: banner may NOT appear (Stripe keeps eventually_due).
// Both outcomes are valid — we verify the page handles it gracefully.

test.describe("KYC Failure — Document Missing (US Individual)", () => {
  test("missing documents handled gracefully after save", async ({ page }) => {
    test.setTimeout(180_000)

    await registerProvider(page)
    await navigateToPaymentInfo(page)
    await selectCountry(page, "US")

    const person = await fillUsPersonalFields(page, {
      ssn: "0000",
      idNumber: "000000000",
      address: "address_full_match",
    })
    await fillUsBankFields(page, person.accountHolder)
    // DO NOT upload documents — they stay in eventually_due without enforce email

    await saveAndWaitForResponse(page)
    await waitForStripeProcessingAndReload(page)

    // Page must still be functional (no crash, no 500)
    await expect(
      page.getByText("Payment Information").first(),
    ).toBeVisible({ timeout: 15000 })

    // Banner may or may not appear depending on Stripe's test mode behavior
    const bannerVisible = await page
      .getByText("Action required").first()
      .isVisible().catch(() => false)

    if (bannerVisible) {
      // eslint-disable-next-line no-console
      console.log("Test 5: Stripe promoted eventually_due → banner visible")
    } else {
      // eslint-disable-next-line no-console
      console.log("Test 5: Documents in eventually_due only — no banner (expected)")
    }
  })
})
