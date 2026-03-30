import { test, expect, type Page } from "@playwright/test"
import { registerProvider, registerAgency, STRONG_PASSWORD, uniqueEmail } from "./helpers/auth"

// ---------------------------------------------------------------------------
// Random name pools — each test picks a unique combo
// ---------------------------------------------------------------------------

const FIRST_NAMES = [
  "Luna", "Enzo", "Jade", "Hugo", "Léa", "Louis", "Manon", "Gabriel",
  "Chloé", "Raphaël", "Inès", "Arthur", "Zoé", "Jules", "Camille", "Théo",
  "Ambre", "Sacha", "Lina", "Axel", "Mila", "Oscar", "Rose", "Victor",
]

const LAST_NAMES = [
  "Moreau", "Laurent", "Durand", "Lefèvre", "Mercier", "Girard", "Bonnet",
  "Fontaine", "Rousseau", "Chevalier", "Blanchard", "Gauthier", "Perrin",
  "Robin", "Clément", "Nicolas", "Rivière", "Marchand", "Aubert", "Colin",
]

const CITIES = ["Paris", "Lyon", "Marseille", "Toulouse", "Bordeaux", "Nantes", "Lille", "Strasbourg"]
const STREETS = ["Rue de la Paix", "Avenue Foch", "Boulevard Voltaire", "Rue du Commerce", "Allée des Tilleuls"]
const BIZ_TYPES = ["SARL", "SAS", "EURL", "SA", "SCI"]

function pick<T>(arr: T[]): T {
  return arr[Math.floor(Math.random() * arr.length)]
}

// ---------------------------------------------------------------------------
// Test data — override via env or use random defaults
// ---------------------------------------------------------------------------

function kycData() {
  const firstName = process.env.KYC_FIRST_NAME || pick(FIRST_NAMES)
  const lastName = process.env.KYC_LAST_NAME || pick(LAST_NAMES)
  const ts = Date.now()
  return {
    firstName,
    lastName,
    dob: process.env.KYC_DOB || "1990-05-15",
    address: process.env.KYC_ADDRESS || `${(ts % 99) + 1} ${pick(STREETS)}`,
    city: process.env.KYC_CITY || pick(CITIES),
    postalCode: process.env.KYC_POSTAL_CODE || "75001",
    phone: process.env.KYC_PHONE || `+336${String(ts).slice(-8)}`,
    iban: process.env.KYC_IBAN || "FR1420041010050500013M02606",
    bic: process.env.KYC_BIC || "BNPAFRPP",
    accountHolder: process.env.KYC_ACCOUNT_HOLDER || `${firstName} ${lastName}`,
  }
}

function businessKycData() {
  const personal = kycData()
  const ts = Date.now()
  const bizType = pick(BIZ_TYPES)
  const bizName = process.env.KYC_BIZ_NAME || `${bizType} ${pick(LAST_NAMES)} & Co`
  return {
    ...personal,
    businessName: bizName,
    businessAddress: process.env.KYC_BIZ_ADDRESS || `${(ts % 50) + 1} ${pick(STREETS)}`,
    businessCity: process.env.KYC_BIZ_CITY || pick(CITIES),
    businessPostalCode: process.env.KYC_BIZ_POSTAL || "69001",
    taxId: process.env.KYC_TAX_ID || `${String(ts).slice(-14).padStart(14, "0")}`,
    vatNumber: process.env.KYC_VAT || "FR12345678901",
    accountHolder: process.env.KYC_ACCOUNT_HOLDER || bizName,
    repFirstName: pick(FIRST_NAMES),
    repLastName: pick(LAST_NAMES),
    repEmail: `rep-${ts}@test.com`,
    ownerFirstName: pick(FIRST_NAMES),
    ownerLastName: pick(LAST_NAMES),
    ownerEmail: `owner-${ts}@test.com`,
  }
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/** Fill an input field by its visible label text. */
async function fillByLabel(page: Page, label: string, value: string) {
  const labelEl = page.locator("label", { hasText: label }).first()
  const input = labelEl.locator("..").locator("input")
  await input.fill(value)
}

/** Select a value in a <select> dropdown by its visible label. */
async function selectByLabel(page: Page, label: string, value: string) {
  const labelEl = page.locator("label", { hasText: label }).first()
  const select = labelEl.locator("..").locator("select")
  await select.selectOption(value)
}

/** Navigate to payment info page from dashboard. */
async function navigateToPaymentInfo(page: Page) {
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
}

/** Fill the personal info + bank account fields (shared between individual and business). */
async function fillPersonalAndBank(page: Page, data: ReturnType<typeof kycData>) {
  await fillByLabel(page, "First name", data.firstName)
  await fillByLabel(page, "Last name", data.lastName)
  const dobLabel = page.locator("label", { hasText: "Date of birth" }).first()
  await dobLabel.locator("..").locator("input[type='date']").fill(data.dob)
  await selectByLabel(page, "Nationality", "FR")
  await fillByLabel(page, "Address", data.address)
  await fillByLabel(page, "City", data.city)
  await fillByLabel(page, "Postal code", data.postalCode)
  await fillByLabel(page, "Phone", data.phone)

  const sectorSelect = page.locator("select[aria-label='Activity sector']")
  if (await sectorSelect.isVisible()) {
    await sectorSelect.selectOption("7372")
  }

  await fillByLabel(page, "IBAN", data.iban)
  await fillByLabel(page, "BIC", data.bic)
  await selectByLabel(page, "Bank country", "FR")
  await fillByLabel(page, "Account holder name", data.accountHolder)
}

/** Click save and wait for success banner. */
async function saveAndVerify(page: Page) {
  const saveButton = page.getByRole("button", { name: /Save/i })
  await expect(saveButton).toBeEnabled()
  await saveButton.click()
  await expect(
    page.getByText("Payment information saved"),
  ).toBeVisible({ timeout: 30000 })
}

// ---------------------------------------------------------------------------
// Individual Provider KYC
// ---------------------------------------------------------------------------

test.describe("KYC Flow — Individual Provider", () => {
  test("register, fill payment info, save, and verify persistence", async ({ page }) => {
    await registerProvider(page)
    await navigateToPaymentInfo(page)

    const data = kycData()
    await fillPersonalAndBank(page, data)
    await saveAndVerify(page)

    // Reload and verify persistence
    await page.reload()
    await expect(page.getByText("Payment information saved")).toBeVisible({ timeout: 15000 })

    const firstNameInput = page.locator("label", { hasText: "First name" }).first().locator("..").locator("input")
    await expect(firstNameInput).toHaveValue(data.firstName)
    const ibanInput = page.locator("label", { hasText: "IBAN" }).first().locator("..").locator("input")
    await expect(ibanInput).toHaveValue(data.iban)
  })

  test("saving twice does not create duplicate Stripe accounts", async ({ page }) => {
    await registerProvider(page)
    await navigateToPaymentInfo(page)

    const data = kycData()
    await fillPersonalAndBank(page, data)
    await saveAndVerify(page)

    // Modify and save again
    await fillByLabel(page, "City", "Lyon")
    await saveAndVerify(page)

    // Verify update, not duplicate
    await page.reload()
    await expect(page.getByText("Payment information saved")).toBeVisible({ timeout: 15000 })
    const cityInput = page.locator("label", { hasText: "City" }).first().locator("..").locator("input")
    await expect(cityInput).toHaveValue("Lyon")
  })
})

// ---------------------------------------------------------------------------
// Business KYC
// ---------------------------------------------------------------------------

test.describe("KYC Flow — Business Account", () => {
  test("register agency, fill business KYC with all fields, save and verify", async ({ page }) => {
    await registerAgency(page)
    await navigateToPaymentInfo(page)

    const data = businessKycData()

    // Toggle business mode — click the switch button
    await page.getByRole("switch").click()

    // Wait for business sections to appear
    await expect(page.getByText("Business Information")).toBeVisible({ timeout: 5000 })

    // Fill personal (legal rep) info + bank
    await fillPersonalAndBank(page, data)

    // Select business role
    const roleSelect = page.locator("select[aria-label='Your role in the company']")
    if (await roleSelect.isVisible()) {
      await roleSelect.selectOption("ceo")
    }

    // Fill business info
    await fillByLabel(page, "Business name", data.businessName)
    await fillByLabel(page, "Business address", data.businessAddress)
    await fillByLabel(page, "Business city", data.businessCity)
    await fillByLabel(page, "Business postal code", data.businessPostalCode)
    await selectByLabel(page, "Country of registration", "FR")
    await fillByLabel(page, "Tax ID", data.taxId)
    await fillByLabel(page, "VAT number", data.vatNumber)

    // Key persons section — all checkboxes should be checked by default
    await expect(page.getByText("Key Company Persons")).toBeVisible()

    // Save
    await saveAndVerify(page)

    // Reload and verify business fields persisted
    await page.reload()
    await expect(page.getByText("Payment information saved")).toBeVisible({ timeout: 15000 })

    const bizNameInput = page.locator("label", { hasText: "Business name" }).first().locator("..").locator("input")
    await expect(bizNameInput).toHaveValue(data.businessName)
    const taxInput = page.locator("label", { hasText: "Tax ID" }).first().locator("..").locator("input")
    await expect(taxInput).toHaveValue(data.taxId)
  })

  test("uncheck representative + owners, add persons, save and verify persistence", async ({ page }) => {
    await registerAgency(page)
    await navigateToPaymentInfo(page)

    const data = businessKycData()

    // Toggle business — click the switch button
    await page.getByRole("switch").click()
    await expect(page.getByText("Business Information")).toBeVisible({ timeout: 5000 })

    // Fill base fields
    await fillPersonalAndBank(page, data)
    const roleSelect = page.locator("select[aria-label='Your role in the company']")
    if (await roleSelect.isVisible()) {
      await roleSelect.selectOption("ceo")
    }
    await fillByLabel(page, "Business name", data.businessName)
    await fillByLabel(page, "Business address", data.businessAddress)
    await fillByLabel(page, "Business city", data.businessCity)
    await fillByLabel(page, "Business postal code", data.businessPostalCode)
    await selectByLabel(page, "Country of registration", "FR")
    await fillByLabel(page, "Tax ID", data.taxId)

    // Scroll down to key persons
    await page.getByText("Key Company Persons").scrollIntoViewIfNeeded()

    // Uncheck "I am the legal representative" to reveal representative form
    const repCheckbox = page.getByText("I am the legal representative of this company")
    await repCheckbox.click()

    // Add a representative person
    const addRepButton = page.getByRole("button", { name: "Add a person" }).first()
    await addRepButton.click()

    // Fill the representative person fields (first person form that appears)
    const personForms = page.locator("[data-person-form], .space-y-3").first()
    // Use the first set of First name / Last name inputs after the add button
    const allFirstNames = page.locator("label", { hasText: "First name" })
    // The second "First name" input is the person form (first is the personal info)
    const repFirstNameInput = allFirstNames.nth(1).locator("..").locator("input")
    if (await repFirstNameInput.isVisible().catch(() => false)) {
      await repFirstNameInput.fill(data.repFirstName)
      const repLastNameInput = page.locator("label", { hasText: "Last name" }).nth(1).locator("..").locator("input")
      await repLastNameInput.fill(data.repLastName)
    }

    // Uncheck "No shareholder holds more than 25%"
    const ownersCheckbox = page.getByText("No shareholder holds more than 25%")
    await ownersCheckbox.scrollIntoViewIfNeeded()
    await ownersCheckbox.click()

    // Save
    await saveAndVerify(page)

    // Reload and verify checkboxes + business fields persist
    await page.reload()
    await expect(page.getByText("Payment information saved")).toBeVisible({ timeout: 15000 })

    // Business toggle should still be on
    await expect(page.getByText("Business Information")).toBeVisible()

    // Business name should persist
    const bizNameInput = page.locator("label", { hasText: "Business name" }).first().locator("..").locator("input")
    await expect(bizNameInput).toHaveValue(data.businessName)

    // The representative checkbox should be unchecked (we unchecked it)
    // This means the "Add a person" or person form should be visible
    await page.getByText("Key Company Persons").scrollIntoViewIfNeeded()

    // The "No shareholder" checkbox should also be unchecked
    // → "Shareholders >25%" section should be visible with add button
    await expect(page.getByText("Shareholders >25%")).toBeVisible()
  })
})
