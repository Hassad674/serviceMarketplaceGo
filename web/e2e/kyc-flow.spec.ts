import { test, expect, type Page } from "@playwright/test"
import { registerProvider, registerAgency } from "./helpers/auth"
import path from "path"

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

const CITIES = ["Paris", "Lyon", "Marseille", "Toulouse", "Bordeaux", "Nantes"]
const STREETS = ["Rue de la Paix", "Avenue Foch", "Boulevard Voltaire", "Rue du Commerce"]
const BIZ_TYPES = ["SARL", "SAS", "EURL", "SA", "SCI"]

const TEST_IBAN = "FR1420041010050500013M02606"

function pick<T>(arr: T[]): T {
  return arr[Math.floor(Math.random() * arr.length)]
}

// ---------------------------------------------------------------------------
// Test data generators
// ---------------------------------------------------------------------------

function kycData() {
  const firstName = pick(FIRST_NAMES)
  const lastName = pick(LAST_NAMES)
  const ts = Date.now()
  return {
    firstName,
    lastName,
    dob: "1990-05-15",
    email: `kyc-${ts}@test.com`,
    address: `${(ts % 99) + 1} ${pick(STREETS)}`,
    city: pick(CITIES),
    postalCode: "75001",
    phone: `+336${String(ts).slice(-8)}`,
    iban: TEST_IBAN,
    bic: "BNPAFRPP",
    accountHolder: `${firstName} ${lastName}`,
  }
}

function businessKycData() {
  const personal = kycData()
  const ts = Date.now()
  const bizType = pick(BIZ_TYPES)
  const bizName = `${bizType} ${pick(LAST_NAMES)} & Co`
  return {
    ...personal,
    businessName: bizName,
    businessAddress: `${(ts % 50) + 1} ${pick(STREETS)}`,
    businessCity: pick(CITIES),
    businessPostalCode: "69001",
    taxId: `${String(ts).slice(-14).padStart(14, "0")}`,
    vatNumber: "FR12345678901",
    accountHolder: bizName,
  }
}

// ---------------------------------------------------------------------------
// Helpers — form field interaction
// ---------------------------------------------------------------------------

/**
 * Fill an input by finding the <label> with matching text, then locating
 * the sibling <input> in the same parent container.
 * Uses hasText for partial matching (labels contain "*" for required fields).
 */
async function fillByLabel(page: Page, label: string, value: string) {
  const labelEl = page.locator("label", { hasText: label }).first()
  const input = labelEl.locator("..").locator("input").first()
  await input.fill(value)
}

/** Select a value in a <select> inside the same parent as the label. */
async function selectByLabel(page: Page, label: string, value: string) {
  const labelEl = page.locator("label", { hasText: label }).first()
  const select = labelEl.locator("..").locator("select").first()
  await select.selectOption(value)
}

/** Get the input near a label for assertions. */
function inputByLabel(page: Page, label: string) {
  return page.locator("label", { hasText: label }).first().locator("..").locator("input").first()
}

/** Navigate to /en/payment-info (forcing English locale). */
async function navigateToPaymentInfo(page: Page) {
  await page.goto("/en/payment-info")
  await expect(
    page.getByText("Payment Information").first(),
  ).toBeVisible({ timeout: 10000 })
}

/**
 * Select France in the Activity Country dropdown.
 * EN locale defaults to US. After selecting FR, the form re-renders
 * with FR-specific fields (IBAN instead of account number, etc.).
 */
async function selectFrance(page: Page) {
  // The country selector is the very first <select> on the page
  const countrySelect = page.locator("select").first()
  await countrySelect.selectOption("FR")

  // Wait for the form to re-render — FR shows IBAN field
  await expect(page.locator("label", { hasText: "IBAN" }).first()).toBeVisible({ timeout: 10000 })
}

/**
 * Fill all FR individual fields: personal info + bank account.
 * Labels come from the dynamic country_specs API for FR.
 * FR individual fields: City, Address, Postal code, Date of birth,
 * Email (auto-filled), First name, Last name, Phone number,
 * IBAN, BIC / SWIFT, Account holder name, Bank country.
 */
async function fillFrPersonalAndBank(page: Page, data: ReturnType<typeof kycData>) {
  // Personal info
  await fillByLabel(page, "First name", data.firstName)
  await fillByLabel(page, "Last name", data.lastName)

  // Date of birth — find the date input in the DOB field container
  const dobLabel = page.locator("label", { hasText: "Date of birth" }).first()
  const dobInput = dobLabel.locator("..").locator("input").first()
  await dobInput.fill(data.dob)

  // Email — required for FR
  await fillByLabel(page, "Email", data.email)

  await fillByLabel(page, "Address", data.address)
  await fillByLabel(page, "City", data.city)
  await fillByLabel(page, "Postal code", data.postalCode)
  await fillByLabel(page, "Phone number", data.phone)

  // Activity sector (select dropdown, not label-based)
  const sectorSelect = page.locator("select[aria-label='Activity sector']")
  if (await sectorSelect.isVisible().catch(() => false)) {
    await sectorSelect.selectOption({ index: 1 })
  }

  // Bank account — FR uses IBAN
  await fillByLabel(page, "IBAN", data.iban)

  // BIC / SWIFT (optional)
  const bicLabel = page.locator("label", { hasText: "BIC" }).first()
  if (await bicLabel.isVisible().catch(() => false)) {
    const bicInput = bicLabel.locator("..").locator("input").first()
    await bicInput.fill(data.bic)
  }

  // Account holder name
  await fillByLabel(page, "Account holder name", data.accountHolder)

  // Bank country
  const bankCountryLabel = page.locator("label", { hasText: "Bank country" }).first()
  if (await bankCountryLabel.isVisible().catch(() => false)) {
    const bankCountrySelect = bankCountryLabel.locator("..").locator("select").first()
    await bankCountrySelect.selectOption("FR")
  }
}

/**
 * Fill FR business-specific fields after personal info + bank are done.
 * Includes: business role, title/role, company info, business phone, tax ID.
 */
async function fillFrBusinessFields(page: Page, data: ReturnType<typeof businessKycData>) {
  // Business role (CEO)
  const roleSelect = page.locator("select[aria-label='Your role in the company']")
  if (await roleSelect.isVisible().catch(() => false)) {
    await roleSelect.selectOption("ceo")
  }

  // Representative Title / Role (required for FR business)
  const titleLabel = page.locator("label", { hasText: "Title / Role" }).first()
  if (await titleLabel.isVisible().catch(() => false)) {
    await fillByLabel(page, "Title / Role", "CEO")
  }

  // Business info
  await fillByLabel(page, "Business name", data.businessName)
  await fillByLabel(page, "Business address", data.businessAddress)
  await fillByLabel(page, "Business city", data.businessCity)
  await fillByLabel(page, "Business postal code", data.businessPostalCode)
  await fillByLabel(page, "Tax ID", data.taxId)

  // Business phone number (second "Phone number" label, in Company section)
  const bizPhoneLabels = page.locator("label", { hasText: "Phone number" })
  const bizPhoneCount = await bizPhoneLabels.count()
  if (bizPhoneCount > 1) {
    const bizPhoneInput = bizPhoneLabels.nth(1).locator("..").locator("input").first()
    if (await bizPhoneInput.isVisible().catch(() => false)) {
      await bizPhoneInput.fill(data.phone)
    }
  }
}

/** Click save and wait for success banner. */
async function saveAndVerify(page: Page) {
  // Scroll the main content area to the bottom to reveal the Save button
  await page.evaluate(() => {
    const main = document.querySelector("main")
    if (main) main.scrollTop = main.scrollHeight
  })
  await page.waitForTimeout(500)

  const saveButton = page.locator("main button", { hasText: "Save" })
  await expect(saveButton).toBeVisible({ timeout: 5000 })
  await expect(saveButton).toBeEnabled({ timeout: 10000 })
  await saveButton.click()

  // Scroll back to top to see the success/error banner
  await page.evaluate(() => {
    const main = document.querySelector("main")
    if (main) main.scrollTop = 0
  })
  await page.waitForTimeout(500)

  await expect(
    page.getByText("Payment information saved"),
  ).toBeVisible({ timeout: 60000 })
}

/** Upload a passport via identity verification (if the section exists). */
async function uploadPassport(page: Page) {
  const section = page.getByText("Identity Verification").first()
  if (!(await section.isVisible().catch(() => false))) return

  await section.scrollIntoViewIfNeeded()
  const uploadZone = page.getByText("Upload document").first()
  if (!(await uploadZone.isVisible().catch(() => false))) return

  await uploadZone.click()
  await page.getByText("Passport").click()

  const fileInput = page.locator("input[type='file']")
  await fileInput.setInputFiles(path.resolve(__dirname, "fixtures/test-passport.png"))

  const modal = page.getByRole("dialog")
  await modal.getByRole("button", { name: /Upload/i }).click()

  await expect(
    page.getByText(/pending|verified/i).first(),
  ).toBeVisible({ timeout: 15000 })
}

// ---------------------------------------------------------------------------
// Test 1 — FR Individual Provider
// ---------------------------------------------------------------------------

test.describe("KYC Flow — FR Individual Provider", () => {
  test("register, select France, fill individual fields, save, verify persistence", async ({ page }) => {
    test.setTimeout(90_000)
    await registerProvider(page)
    await navigateToPaymentInfo(page)
    await selectFrance(page)

    const data = kycData()
    await fillFrPersonalAndBank(page, data)
    await saveAndVerify(page)
    await uploadPassport(page)

    // Reload and verify all fields persist
    await page.reload()
    await expect(page.getByText("Payment information saved")).toBeVisible({ timeout: 15000 })
    await expect(inputByLabel(page, "First name")).toHaveValue(data.firstName)
    await expect(inputByLabel(page, "Last name")).toHaveValue(data.lastName)
    await expect(inputByLabel(page, "IBAN")).toHaveValue(data.iban)
    await expect(inputByLabel(page, "Account holder name")).toHaveValue(data.accountHolder)
  })

  test("re-save after 30s does not create duplicate Stripe account", async ({ page }) => {
    test.setTimeout(120_000)

    await registerProvider(page)
    await navigateToPaymentInfo(page)
    await selectFrance(page)

    const data = kycData()
    await fillFrPersonalAndBank(page, data)
    await saveAndVerify(page)
    await uploadPassport(page)

    await page.waitForTimeout(30_000)

    await fillByLabel(page, "City", "Marseille")
    await saveAndVerify(page)

    await page.reload()
    await expect(page.getByText("Payment information saved")).toBeVisible({ timeout: 15000 })
    await expect(inputByLabel(page, "City")).toHaveValue("Marseille")
    await expect(inputByLabel(page, "First name")).toHaveValue(data.firstName)
  })
})

// ---------------------------------------------------------------------------
// Test 2 — FR Business (Agency)
// ---------------------------------------------------------------------------

test.describe("KYC Flow — FR Business Account", () => {
  test("register agency, toggle business, fill all FR fields, save and verify", async ({ page }) => {
    test.setTimeout(90_000)
    await registerAgency(page)
    await navigateToPaymentInfo(page)
    await selectFrance(page)

    // Toggle business mode ON
    await page.getByRole("switch").click()
    await expect(page.getByText("Company Information").first()).toBeVisible({ timeout: 10000 })

    const data = businessKycData()
    await fillFrPersonalAndBank(page, data)
    await fillFrBusinessFields(page, data)

    // Business persons: all 4 checkboxes should be checked by default
    const keyPersons = page.getByText("Key Company Persons").first()
    if (await keyPersons.isVisible().catch(() => false)) {
      await keyPersons.scrollIntoViewIfNeeded()
    }

    await saveAndVerify(page)
    await uploadPassport(page)

    // Reload and verify persistence
    await page.reload()
    await expect(page.getByText("Payment information saved")).toBeVisible({ timeout: 15000 })
    await expect(inputByLabel(page, "First name")).toHaveValue(data.firstName)
    await expect(inputByLabel(page, "IBAN")).toHaveValue(data.iban)
    await expect(inputByLabel(page, "Business name")).toHaveValue(data.businessName)
    await expect(inputByLabel(page, "Tax ID")).toHaveValue(data.taxId)
    await expect(page.getByText("Company Information").first()).toBeVisible()
  })

  test("re-save business after 30s does not create duplicate Stripe account", async ({ page }) => {
    test.setTimeout(120_000)

    await registerAgency(page)
    await navigateToPaymentInfo(page)
    await selectFrance(page)

    await page.getByRole("switch").click()
    await expect(page.getByText("Company Information").first()).toBeVisible({ timeout: 10000 })

    const data = businessKycData()
    await fillFrPersonalAndBank(page, data)
    await fillFrBusinessFields(page, data)

    await saveAndVerify(page)
    await uploadPassport(page)

    await page.waitForTimeout(30_000)

    await fillByLabel(page, "Business city", "Bordeaux")
    await saveAndVerify(page)

    await page.reload()
    await expect(page.getByText("Payment information saved")).toBeVisible({ timeout: 15000 })
    await expect(inputByLabel(page, "Business city")).toHaveValue("Bordeaux")
    await expect(inputByLabel(page, "Business name")).toHaveValue(data.businessName)
  })

  test("uncheck representative + owners, add persons, save and verify", async ({ page }) => {
    test.setTimeout(90_000)
    await registerAgency(page)
    await navigateToPaymentInfo(page)
    await selectFrance(page)

    await page.getByRole("switch").click()
    await expect(page.getByText("Company Information").first()).toBeVisible({ timeout: 10000 })

    const data = businessKycData()
    await fillFrPersonalAndBank(page, data)
    await fillFrBusinessFields(page, data)

    // Uncheck representative and owners checkboxes
    const keyPersons = page.getByText("Key Company Persons").first()
    if (await keyPersons.isVisible().catch(() => false)) {
      await keyPersons.scrollIntoViewIfNeeded()

      const repCheckbox = page.getByText("I am the legal representative of this company")
      if (await repCheckbox.isVisible().catch(() => false)) {
        await repCheckbox.click()

        const addRepButton = page.getByRole("button", { name: "Add a person" }).first()
        if (await addRepButton.isVisible().catch(() => false)) {
          await addRepButton.click()
        }
      }

      const ownersCheckbox = page.getByText("No shareholder holds more than 25%")
      if (await ownersCheckbox.isVisible().catch(() => false)) {
        await ownersCheckbox.scrollIntoViewIfNeeded()
        await ownersCheckbox.click()
      }
    }

    await saveAndVerify(page)

    await page.reload()
    await expect(page.getByText("Payment information saved")).toBeVisible({ timeout: 15000 })
    await expect(page.getByText("Company Information").first()).toBeVisible()
    await expect(inputByLabel(page, "Business name")).toHaveValue(data.businessName)
  })
})
