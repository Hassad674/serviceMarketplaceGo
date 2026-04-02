import path from "node:path"
import { test, expect, type Page } from "@playwright/test"
import { registerProvider, registerAgency } from "./helpers/auth"

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
    businessPhone: `+337${String(ts).slice(-8)}`,
    taxId: `${String(ts).slice(-14).padStart(14, "0")}`,
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
  await input.scrollIntoViewIfNeeded()
  await input.fill(value)
}

/** Select a value in a <select> inside the same parent as the label. */
async function selectByLabel(page: Page, label: string, value: string) {
  const labelEl = page.locator("label", { hasText: label }).first()
  const select = labelEl.locator("..").locator("select").first()
  await select.scrollIntoViewIfNeeded()
  await select.selectOption(value)
}

/**
 * Upload a document for each visible document upload zone on the page.
 *
 * Document zones are identified by a label like "Official identity document"
 * followed by a dashed-border button containing "Click to upload" text.
 * Clicking that button opens an UploadModal (portaled to body). Inside the
 * modal we set the file on the hidden <input type="file"> and click Upload.
 * We then wait for the zone to switch to a green "Document uploaded" badge.
 */
async function uploadDocuments(page: Page, expectedCount?: number) {
  const fixturePath = path.resolve(__dirname, "fixtures", "test-passport.png")

  // Document upload zones are <button> elements inside the form
  // with the "Click to upload" text rendered by DocumentUploadField.
  const uploadButtons = page.locator("button", { hasText: "Click to upload" })
  const count = await uploadButtons.count()

  // Sanity check: if caller specifies expected count, verify it
  if (expectedCount !== undefined && count !== expectedCount) {
    throw new Error(`Expected ${expectedCount} document upload zones but found ${count}`)
  }

  for (let i = 0; i < count; i++) {
    // Re-query each time because the DOM mutates after each upload
    // (the uploaded zone switches from button to a green badge div)
    const btn = page.locator("button", { hasText: "Click to upload" }).first()
    await btn.scrollIntoViewIfNeeded()
    await btn.click()

    // Wait for the upload modal to appear (portaled to body)
    const modal = page.locator("div[role='dialog']")
    await expect(modal).toBeVisible({ timeout: 5000 })

    // Set the file on the hidden file input inside the modal
    const fileInput = modal.locator("input[type='file']")
    await fileInput.setInputFiles(fixturePath)

    // Wait for file preview to appear (the file name or preview image)
    await expect(modal.locator("text=test-passport.png")).toBeVisible({ timeout: 5000 })

    // Click the Upload button inside the modal (the action button, not the close/cancel)
    const uploadBtn = modal.locator("button", { hasText: "Upload" }).last()
    await uploadBtn.click()

    // Wait for the modal to close
    await expect(modal).toBeHidden({ timeout: 15000 })

    // Wait for the "Document uploaded" green badge to appear for this document
    // Each successful upload converts the button to a green badge with this text
    await expect(
      page.getByText("Document uploaded").nth(i),
    ).toBeVisible({ timeout: 15000 })
  }
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
  // The country selector has the "Activity Country" heading.
  // The <select> is inside that section.
  const countrySection = page.locator("section, div").filter({ hasText: "Activity Country" }).first()
  const countrySelect = countrySection.locator("select").first()
  await countrySelect.selectOption("FR")

  // Wait for the form to re-render with FR-specific fields
  await expect(page.locator("label", { hasText: "IBAN" }).first()).toBeVisible({ timeout: 10000 })
}

/**
 * Fill all FR individual fields: personal info + bank account.
 * Labels come from the dynamic country_specs API for FR.
 * FR individual fields: City, Address, Postal code, Date of birth,
 * Email, First name, Last name, Phone number,
 * IBAN, BIC / SWIFT, Account holder name, Bank country.
 */
async function fillFrPersonalAndBank(page: Page, data: ReturnType<typeof kycData>) {
  // Personal info — fields rendered by DynamicSection
  await fillByLabel(page, "First name", data.firstName)
  await fillByLabel(page, "Last name", data.lastName)

  // Date of birth — rendered as a date input
  const dobLabel = page.locator("label", { hasText: "Date of birth" }).first()
  const dobInput = dobLabel.locator("..").locator("input").first()
  await dobInput.scrollIntoViewIfNeeded()
  await dobInput.fill(data.dob)

  // Email — required for FR
  await fillByLabel(page, "Email", data.email)

  await fillByLabel(page, "Address", data.address)
  await fillByLabel(page, "City", data.city)
  await fillByLabel(page, "Postal code", data.postalCode)
  await fillByLabel(page, "Phone number", data.phone)

  // Activity sector (select dropdown with aria-label)
  const sectorSelect = page.locator("select[aria-label='Activity sector']")
  if (await sectorSelect.isVisible().catch(() => false)) {
    await sectorSelect.selectOption({ index: 1 })
  }

  // Bank account section — FR uses IBAN
  await fillByLabel(page, "IBAN", data.iban)

  // BIC / SWIFT (optional but fill it)
  const bicLabel = page.locator("label", { hasText: "BIC" }).first()
  if (await bicLabel.isVisible().catch(() => false)) {
    const bicInput = bicLabel.locator("..").locator("input").first()
    await bicInput.scrollIntoViewIfNeeded()
    await bicInput.fill(data.bic)
  }

  // Account holder name
  await fillByLabel(page, "Account holder name", data.accountHolder)

  // Bank country — a CountrySelect rendered as a <select>
  const bankCountryLabel = page.locator("label", { hasText: "Bank country" }).first()
  if (await bankCountryLabel.isVisible().catch(() => false)) {
    const bankCountrySelect = bankCountryLabel.locator("..").locator("select").first()
    await bankCountrySelect.scrollIntoViewIfNeeded()
    await bankCountrySelect.selectOption("FR")
  }
}

/**
 * Fill FR business-specific fields after personal info + bank are done.
 * Includes: business role, representative title/role, company info
 * (name, address, city, postal code, phone, tax ID).
 */
async function fillFrBusinessFields(page: Page, data: ReturnType<typeof businessKycData>) {
  // Business role dropdown (aria-label "Your role in the company")
  const roleSelect = page.locator("select[aria-label='Your role in the company']")
  if (await roleSelect.isVisible().catch(() => false)) {
    await roleSelect.scrollIntoViewIfNeeded()
    await roleSelect.selectOption("ceo")
  }

  // Representative Title / Role — required field in the Legal Representative section
  const titleLabel = page.locator("label", { hasText: "Title / Role" }).first()
  if (await titleLabel.isVisible().catch(() => false)) {
    await fillByLabel(page, "Title / Role", "CEO")
  }

  // Company Information fields — rendered by DynamicSection with "companyInfo" title
  await fillByLabel(page, "Business name", data.businessName)
  await fillByLabel(page, "Business address", data.businessAddress)
  await fillByLabel(page, "Business city", data.businessCity)
  await fillByLabel(page, "Business postal code", data.businessPostalCode)
  await fillByLabel(page, "Tax ID", data.taxId)

  // Company phone number — the DynamicSection renders "Phone number" for company.phone
  // This is the second "Phone number" label on the page (first is personal, second is company)
  const phoneLabels = page.locator("label", { hasText: "Phone number" })
  const phoneCount = await phoneLabels.count()
  if (phoneCount > 1) {
    const companyPhoneInput = phoneLabels.nth(1).locator("..").locator("input").first()
    if (await companyPhoneInput.isVisible().catch(() => false)) {
      await companyPhoneInput.scrollIntoViewIfNeeded()
      await companyPhoneInput.fill(data.businessPhone)
    }
  }
}

/** Click save and wait for success banner. */
async function saveAndVerify(page: Page) {
  // Scroll to the very bottom to reveal the Save button
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

  // Listen for the save API response before clicking
  const responsePromise = page.waitForResponse(
    (resp) => resp.url().includes("/payment-info") && resp.request().method() === "PUT",
    { timeout: 60000 },
  )

  await saveButton.click()

  // Wait for the API response
  await responsePromise

  // Scroll back to top to see the success banner
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
// Test 1 — FR Individual Provider
// ---------------------------------------------------------------------------

test.describe("KYC Flow — FR Individual Provider", () => {
  test("register, select France, fill individual fields, save, verify persistence", async ({ page }) => {
    test.setTimeout(120_000)
    await registerProvider(page)
    await navigateToPaymentInfo(page)
    await selectFrance(page)

    const data = kycData()
    await fillFrPersonalAndBank(page, data)

    // Upload required identity documents (2 zones for FR individual)
    await uploadDocuments(page, 2)

    await saveAndVerify(page)

    // Reload and verify all fields persist
    await page.reload()
    await expect(page.getByText("Payment information saved")).toBeVisible({ timeout: 15000 })
    await expect(inputByLabel(page, "First name")).toHaveValue(data.firstName)
    await expect(inputByLabel(page, "Last name")).toHaveValue(data.lastName)
    await expect(inputByLabel(page, "IBAN")).toHaveValue(data.iban)
    await expect(inputByLabel(page, "Account holder name")).toHaveValue(data.accountHolder)
  })
})

// ---------------------------------------------------------------------------
// Test 2 — FR Business (Agency)
// ---------------------------------------------------------------------------

test.describe("KYC Flow — FR Business Account", () => {
  test("register agency, toggle business, fill all FR fields, save and verify", async ({ page }) => {
    test.setTimeout(120_000)
    await registerAgency(page)
    await navigateToPaymentInfo(page)
    await selectFrance(page)

    // Toggle business mode ON
    await page.getByRole("switch").click()

    // Wait for the "Company Information" section to appear (dynamic rendering)
    await expect(page.getByText("Company Information").first()).toBeVisible({ timeout: 10000 })

    const data = businessKycData()

    // Fill representative (personal) info + bank
    await fillFrPersonalAndBank(page, data)

    // Fill business-specific fields
    await fillFrBusinessFields(page, data)

    // Verify business persons checkboxes are visible and checked by default.
    // There are 3 checkboxes (NO representative checkbox):
    // 1. "The representative is the sole director" (checked)
    // 2. "No shareholder holds more than 25%" (checked)
    // 3. "The representative is the sole executive" (checked)
    const keyPersonsSection = page.getByText("Key Company Persons").first()
    if (await keyPersonsSection.isVisible().catch(() => false)) {
      await keyPersonsSection.scrollIntoViewIfNeeded()

      const directorCheckbox = page.locator("label").filter({ hasText: "The representative is the sole director" }).locator("input[type='checkbox']")
      await expect(directorCheckbox).toBeChecked()

      const ownersCheckbox = page.locator("label").filter({ hasText: "No shareholder holds more than 25%" }).locator("input[type='checkbox']")
      await expect(ownersCheckbox).toBeChecked()

      const executiveCheckbox = page.locator("label").filter({ hasText: "The representative is the sole executive" }).locator("input[type='checkbox']")
      await expect(executiveCheckbox).toBeChecked()
    }

    // Upload required documents (3 zones for FR business: company + rep identity + rep additional)
    await uploadDocuments(page, 3)

    await saveAndVerify(page)

    // Reload and verify persistence
    await page.reload()
    await expect(page.getByText("Payment information saved")).toBeVisible({ timeout: 15000 })
    await expect(inputByLabel(page, "First name")).toHaveValue(data.firstName)
    await expect(inputByLabel(page, "IBAN")).toHaveValue(data.iban)
    await expect(inputByLabel(page, "Business name")).toHaveValue(data.businessName)
    await expect(inputByLabel(page, "Tax ID")).toHaveValue(data.taxId)
    await expect(page.getByText("Company Information").first()).toBeVisible()
  })
})
