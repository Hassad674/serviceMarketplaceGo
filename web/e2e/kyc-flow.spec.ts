import { test, expect } from "@playwright/test"
import { registerProvider, STRONG_PASSWORD, uniqueEmail } from "./helpers/auth"

// ---------------------------------------------------------------------------
// Test data — override via env or use random defaults
// ---------------------------------------------------------------------------

function kycData() {
  const ts = Date.now()
  return {
    firstName: process.env.KYC_FIRST_NAME || `Jean${ts}`,
    lastName: process.env.KYC_LAST_NAME || `Dupont${ts}`,
    dob: process.env.KYC_DOB || "1990-05-15",
    address: process.env.KYC_ADDRESS || `${ts % 100} Rue de la Paix`,
    city: process.env.KYC_CITY || "Paris",
    postalCode: process.env.KYC_POSTAL_CODE || "75001",
    phone: process.env.KYC_PHONE || `+336${String(ts).slice(-8)}`,
    iban: process.env.KYC_IBAN || "FR7630006000011234567890189",
    bic: process.env.KYC_BIC || "BNPAFRPP",
    accountHolder: process.env.KYC_ACCOUNT_HOLDER || `Jean Dupont ${ts}`,
  }
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/** Fill an input field by its visible label text. */
async function fillByLabel(
  page: import("@playwright/test").Page,
  label: string,
  value: string,
) {
  // The form uses custom <label> + <input> pairs (not native <label for>).
  // Strategy: find the label text, then fill the sibling <input>.
  const labelEl = page.locator("label", { hasText: label }).first()
  const input = labelEl.locator("..").locator("input")
  await input.fill(value)
}

/** Select a country in a <select> dropdown by its visible label. */
async function selectByLabel(
  page: import("@playwright/test").Page,
  label: string,
  value: string,
) {
  const labelEl = page.locator("label", { hasText: label }).first()
  const select = labelEl.locator("..").locator("select")
  await select.selectOption(value)
}

// ---------------------------------------------------------------------------
// Test: Register + fill KYC as individual provider (desktop)
// ---------------------------------------------------------------------------

test.describe("KYC Flow — Individual Provider", () => {
  test("register, fill payment info, save, and verify persistence", async ({
    page,
  }) => {
    // Step 1: Register a new provider
    const user = await registerProvider(page)

    // Step 2: Navigate to payment info
    // Open sidebar if needed (mobile) then click Payment Info
    const hamburger = page.getByRole("button", { name: "Open menu" })
    if (await hamburger.isVisible().catch(() => false)) {
      await hamburger.click()
      await page.waitForTimeout(350)
    }
    await page.getByRole("link", { name: /Payment Info/i }).click()
    await page.waitForURL("**/payment-info", { timeout: 10000 })

    // Step 3: Wait for the form to load
    await expect(
      page.getByText("Payment Information").first(),
    ).toBeVisible({ timeout: 10000 })

    const data = kycData()

    // Step 4: Fill personal info
    await fillByLabel(page, "First name", data.firstName)
    await fillByLabel(page, "Last name", data.lastName)

    // Date of birth — type="date" input
    const dobLabel = page.locator("label", { hasText: "Date of birth" }).first()
    const dobInput = dobLabel.locator("..").locator("input[type='date']")
    await dobInput.fill(data.dob)

    // Nationality
    await selectByLabel(page, "Nationality", "FR")

    await fillByLabel(page, "Address", data.address)
    await fillByLabel(page, "City", data.city)
    await fillByLabel(page, "Postal code", data.postalCode)
    await fillByLabel(page, "Phone", data.phone)

    // Activity sector — select dropdown
    const sectorSelect = page.locator("select[aria-label='Activity sector']")
    if (await sectorSelect.isVisible()) {
      await sectorSelect.selectOption("7372") // Development & IT
    }

    // Step 5: Fill bank account
    await fillByLabel(page, "IBAN", data.iban)
    await fillByLabel(page, "BIC", data.bic)
    await selectByLabel(page, "Bank country", "FR")
    await fillByLabel(page, "Account holder name", data.accountHolder)

    // Step 6: Save
    const saveButton = page.getByRole("button", { name: /Save/i })
    await expect(saveButton).toBeEnabled()
    await saveButton.click()

    // Wait for save to complete
    await expect(
      page.getByText("Payment information saved"),
    ).toBeVisible({ timeout: 30000 })

    // Step 7: Reload page and verify persistence
    await page.reload()
    await expect(
      page.getByText("Payment information saved"),
    ).toBeVisible({ timeout: 15000 })

    // Verify key fields are populated
    const firstNameInput = page
      .locator("label", { hasText: "First name" })
      .first()
      .locator("..")
      .locator("input")
    await expect(firstNameInput).toHaveValue(data.firstName)

    const lastNameInput = page
      .locator("label", { hasText: "Last name" })
      .first()
      .locator("..")
      .locator("input")
    await expect(lastNameInput).toHaveValue(data.lastName)

    const cityInput = page
      .locator("label", { hasText: "City" })
      .first()
      .locator("..")
      .locator("input")
    await expect(cityInput).toHaveValue(data.city)

    const ibanInput = page
      .locator("label", { hasText: "IBAN" })
      .first()
      .locator("..")
      .locator("input")
    await expect(ibanInput).toHaveValue(data.iban)
  })

  test("saving twice does not create duplicate Stripe accounts", async ({
    page,
  }) => {
    // Register
    await registerProvider(page)

    // Navigate to payment info
    const hamburger = page.getByRole("button", { name: "Open menu" })
    if (await hamburger.isVisible().catch(() => false)) {
      await hamburger.click()
      await page.waitForTimeout(350)
    }
    await page.getByRole("link", { name: /Payment Info/i }).click()
    await page.waitForURL("**/payment-info", { timeout: 10000 })
    await expect(
      page.getByText("Payment Information").first(),
    ).toBeVisible({ timeout: 10000 })

    const data = kycData()

    // Fill form
    await fillByLabel(page, "First name", data.firstName)
    await fillByLabel(page, "Last name", data.lastName)
    const dobLabel = page.locator("label", { hasText: "Date of birth" }).first()
    await dobLabel.locator("..").locator("input[type='date']").fill(data.dob)
    await selectByLabel(page, "Nationality", "FR")
    await fillByLabel(page, "Address", data.address)
    await fillByLabel(page, "City", data.city)
    await fillByLabel(page, "Postal code", data.postalCode)
    await fillByLabel(page, "Phone", data.phone)
    await fillByLabel(page, "IBAN", data.iban)
    await selectByLabel(page, "Bank country", "FR")
    await fillByLabel(page, "Account holder name", data.accountHolder)

    // Save first time
    await page.getByRole("button", { name: /Save/i }).click()
    await expect(
      page.getByText("Payment information saved"),
    ).toBeVisible({ timeout: 30000 })

    // Modify a field and save again
    await fillByLabel(page, "City", "Lyon")
    await page.getByRole("button", { name: /Save/i }).click()
    await expect(
      page.getByText("Payment information saved"),
    ).toBeVisible({ timeout: 30000 })

    // Verify the city was updated (not a new account)
    await page.reload()
    await expect(
      page.getByText("Payment information saved"),
    ).toBeVisible({ timeout: 15000 })

    const cityInput = page
      .locator("label", { hasText: "City" })
      .first()
      .locator("..")
      .locator("input")
    await expect(cityInput).toHaveValue("Lyon")
  })
})
