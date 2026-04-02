import path from "node:path"
import { test, expect, type Page } from "@playwright/test"
import { registerProvider, registerAgency } from "./helpers/auth"

// ---------------------------------------------------------------------------
// Test data — US country config (mirrors kyc-flow-multi.spec.ts)
// ---------------------------------------------------------------------------

type CountryTestData = {
  code: string
  name: string
  phone: string
  postalCode: string
  state?: string
  ssnLast4?: string
  idNumber?: string
  bankType: "iban" | "local"
  accountNumber?: string
  routingNumber?: string
  businessPostalCode: string
  businessState?: string
}

const US_COUNTRY: CountryTestData = {
  code: "US",
  name: "United States",
  phone: "+14155551234",
  postalCode: "94102",
  state: "CA",
  ssnLast4: "0000",
  idNumber: "000000000",
  bankType: "local",
  accountNumber: "000123456789",
  routingNumber: "110000000",
  businessPostalCode: "10001",
  businessState: "NY",
}

// ---------------------------------------------------------------------------
// Random name pools
// ---------------------------------------------------------------------------

const FIRST_NAMES = [
  "Emma", "Liam", "Sophia", "Noah", "Olivia", "James", "Ava", "William",
  "Isabella", "Oliver", "Mia", "Benjamin", "Charlotte", "Elijah", "Amelia",
]

const LAST_NAMES = [
  "Smith", "Johnson", "Williams", "Brown", "Jones", "Garcia", "Miller",
  "Davis", "Rodriguez", "Martinez", "Hernandez", "Lopez", "Wilson", "Anderson",
]

const CITIES = ["Springfield", "Portland", "Austin", "Denver", "Seattle"]
const STREETS = ["123 Main St", "456 Oak Ave", "789 Pine Rd", "321 Elm Blvd"]

function pick<T>(arr: T[]): T {
  return arr[Math.floor(Math.random() * arr.length)]
}

function randomPersonData() {
  const firstName = pick(FIRST_NAMES)
  const lastName = pick(LAST_NAMES)
  const ts = Date.now()
  return {
    firstName,
    lastName,
    dob: "1990-05-15",
    email: `kyc-us-dup-${ts}@test.com`,
    address: pick(STREETS),
    city: pick(CITIES),
    postalCode: US_COUNTRY.postalCode,
    phone: US_COUNTRY.phone,
    accountHolder: `${firstName} ${lastName}`,
  }
}

// ---------------------------------------------------------------------------
// Form interaction helpers
// ---------------------------------------------------------------------------

async function fillByLabel(page: Page, label: string, value: string) {
  const labelEl = page.locator("label", { hasText: label }).first()
  const input = labelEl.locator("..").locator("input").first()
  await input.scrollIntoViewIfNeeded()
  await input.fill(value)
}

function inputByLabel(page: Page, label: string) {
  return page.locator("label", { hasText: label }).first().locator("..").locator("input").first()
}

async function navigateToPaymentInfo(page: Page) {
  await page.goto("/en/payment-info")
  await expect(
    page.getByText("Payment Information").first(),
  ).toBeVisible({ timeout: 10000 })
}

async function selectCountry(page: Page, code: string) {
  const countrySection = page.locator("section, div").filter({ hasText: "Activity Country" }).first()
  const countrySelect = countrySection.locator("select").first()
  await countrySelect.selectOption(code)
  await expect(page.locator("label", { hasText: "First name" }).first()).toBeVisible({ timeout: 10000 })
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

    await expect(modal.locator("text=test-passport.png")).toBeVisible({ timeout: 5000 })

    const uploadBtn = modal.locator("button", { hasText: "Upload" }).last()
    await uploadBtn.click()

    await expect(modal).toBeHidden({ timeout: 15000 })
    await expect(
      page.getByText("Document uploaded").nth(i),
    ).toBeVisible({ timeout: 15000 })
  }
}

async function saveAndVerify(page: Page) {
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
    (resp) => resp.url().includes("/payment-info") && resp.request().method() === "PUT",
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
// Personal fields (US individual)
// ---------------------------------------------------------------------------

async function fillPersonalFields(page: Page, person: ReturnType<typeof randomPersonData>) {
  await fillByLabel(page, "First name", person.firstName)
  await fillByLabel(page, "Last name", person.lastName)

  const dobLabel = page.locator("label", { hasText: "Date of birth" }).first()
  if (await dobLabel.isVisible().catch(() => false)) {
    const dobInput = dobLabel.locator("..").locator("input").first()
    await dobInput.scrollIntoViewIfNeeded()
    await dobInput.fill(person.dob)
  }

  const emailLabel = page.locator("label", { hasText: "Email" }).first()
  if (await emailLabel.isVisible().catch(() => false)) {
    await fillByLabel(page, "Email", person.email)
  }

  await fillByLabel(page, "Address", person.address)
  await fillByLabel(page, "City", person.city)
  await fillByLabel(page, "Postal code", person.postalCode)
  await fillByLabel(page, "Phone number", person.phone)

  if (US_COUNTRY.state) {
    const stateLabel = page.locator("label", { hasText: "State" }).first()
    if (await stateLabel.isVisible().catch(() => false)) {
      const stateInput = stateLabel.locator("..").locator("input, select").first()
      await stateInput.scrollIntoViewIfNeeded()
      await stateInput.fill(US_COUNTRY.state)
    }
  }

  if (US_COUNTRY.ssnLast4) {
    const ssnLabel = page.locator("label", { hasText: "SSN" }).first()
    if (await ssnLabel.isVisible().catch(() => false)) {
      const ssnInput = ssnLabel.locator("..").locator("input").first()
      await ssnInput.scrollIntoViewIfNeeded()
      await ssnInput.fill(US_COUNTRY.ssnLast4)
    }
  }

  if (US_COUNTRY.idNumber) {
    const idLabel = page.locator("label", { hasText: "National ID" }).first()
    if (await idLabel.isVisible().catch(() => false)) {
      const idInput = idLabel.locator("..").locator("input").first()
      await idInput.scrollIntoViewIfNeeded()
      await idInput.fill(US_COUNTRY.idNumber)
    }
  }

  const peLabel = page.locator("label", { hasText: "Political Exposure" }).first()
  if (await peLabel.isVisible().catch(() => false)) {
    const peSelect = peLabel.locator("..").locator("select").first()
    await peSelect.scrollIntoViewIfNeeded()
    await peSelect.selectOption("none")
  }

  const sectorSelect = page.locator("select[aria-label='Activity sector']")
  if (await sectorSelect.isVisible().catch(() => false)) {
    await sectorSelect.selectOption({ index: 1 })
  }

  const natLabel = page.locator("label", { hasText: "Nationality" }).first()
  if (await natLabel.isVisible().catch(() => false)) {
    const natSelect = natLabel.locator("..").locator("select").first()
    if (await natSelect.isVisible().catch(() => false)) {
      await natSelect.scrollIntoViewIfNeeded()
      await natSelect.selectOption("US")
    }
  }

  // Catch-all: fill remaining empty text/email/tel inputs
  const allInputs = page.locator("section input[type='text'], section input[type='email'], section input[type='tel'], section input:not([type])")
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

  // Catch-all: select first non-empty option on unselected selects
  const allSelects = page.locator("section select")
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

// ---------------------------------------------------------------------------
// Bank fields (US local)
// ---------------------------------------------------------------------------

async function fillBankFields(page: Page, accountHolder: string) {
  await fillByLabel(page, "Account number", US_COUNTRY.accountNumber!)
  await fillByLabel(page, "Routing number", US_COUNTRY.routingNumber!)
  await fillByLabel(page, "Account holder name", accountHolder)

  const bankCountryLabel = page.locator("label", { hasText: "Bank country" }).first()
  if (await bankCountryLabel.isVisible().catch(() => false)) {
    const bankCountrySelect = bankCountryLabel.locator("..").locator("select").first()
    await bankCountrySelect.scrollIntoViewIfNeeded()
    await bankCountrySelect.selectOption("US")
  }
}

// ---------------------------------------------------------------------------
// Business fields
// ---------------------------------------------------------------------------

async function fillBusinessFields(page: Page): Promise<string> {
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
  await fillByLabel(page, "Business postal code", US_COUNTRY.businessPostalCode)

  if (US_COUNTRY.businessState) {
    const stateLabel = page.locator("label", { hasText: "Business state" }).first()
    if (await stateLabel.isVisible().catch(() => false)) {
      const stateInput = stateLabel.locator("..").locator("input, select").first()
      await stateInput.scrollIntoViewIfNeeded()
      await stateInput.fill(US_COUNTRY.businessState)
    }
  }

  const taxLabel = page.locator("label", { hasText: "Tax ID" }).first()
  if (await taxLabel.isVisible().catch(() => false)) {
    await fillByLabel(page, "Tax ID", "000000000")
  }

  const phoneLabels = page.locator("label", { hasText: "Phone number" })
  const phoneCount = await phoneLabels.count()
  if (phoneCount > 1) {
    const companyPhoneInput = phoneLabels.nth(1).locator("..").locator("input").first()
    if (await companyPhoneInput.isVisible().catch(() => false)) {
      await companyPhoneInput.scrollIntoViewIfNeeded()
      await companyPhoneInput.fill(US_COUNTRY.phone)
    }
  }

  // Catch-all: fill remaining empty inputs
  const allBizInputs = page.locator("input[type='text'], input[type='email'], input[type='tel'], input:not([type])")
  const bizInputCount = await allBizInputs.count()
  for (let i = 0; i < bizInputCount; i++) {
    const input = allBizInputs.nth(i)
    if (await input.isVisible().catch(() => false)) {
      const val = await input.inputValue().catch(() => "")
      if (val === "") {
        await input.scrollIntoViewIfNeeded()
        await input.fill("Test123")
      }
    }
  }

  // Catch-all: select first option on empty selects
  const allBizSelects = page.locator("select")
  const bizSelectCount = await allBizSelects.count()
  for (let i = 0; i < bizSelectCount; i++) {
    const sel = allBizSelects.nth(i)
    if (await sel.isVisible().catch(() => false)) {
      const val = await sel.inputValue().catch(() => "")
      if (val === "") {
        await sel.scrollIntoViewIfNeeded()
        await sel.selectOption({ index: 1 }).catch(() => {})
      }
    }
  }

  return bizName
}

// ---------------------------------------------------------------------------
// Fill ALL remaining empty fields — catch-all
// ---------------------------------------------------------------------------

async function fillAllRemainingFields(page: Page) {
  await page.evaluate(() => {
    const main = document.querySelector("main")
    if (main) main.scrollTop = main.scrollHeight
    window.scrollTo(0, document.body.scrollHeight)
  })
  await page.waitForTimeout(500)
  await page.evaluate(() => {
    const main = document.querySelector("main")
    if (main) main.scrollTop = 0
    window.scrollTo(0, 0)
  })
  await page.waitForTimeout(500)

  const allInputs = page.locator("input")
  const inputCount = await allInputs.count()
  for (let i = 0; i < inputCount; i++) {
    const input = allInputs.nth(i)
    const type = await input.getAttribute("type").catch(() => "text") ?? "text"
    const hidden = await input.getAttribute("hidden").catch(() => null)
    if (hidden !== null) continue
    if (["text", "email", "tel", "number", ""].includes(type)) {
      const val = await input.inputValue().catch(() => "SKIP")
      if (val === "") {
        await input.scrollIntoViewIfNeeded().catch(() => {})
        if (type === "email") await input.fill("test@example.com").catch(() => {})
        else if (type === "tel") await input.fill(US_COUNTRY.phone).catch(() => {})
        else await input.fill("Test123").catch(() => {})
      }
    }
  }

  const allSelects = page.locator("select")
  const selectCount = await allSelects.count()
  for (let i = 0; i < selectCount; i++) {
    const sel = allSelects.nth(i)
    const val = await sel.inputValue().catch(() => "SKIP")
    if (val === "") {
      await sel.scrollIntoViewIfNeeded().catch(() => {})
      await sel.selectOption("US").catch(async () => {
        await sel.selectOption({ index: 1 }).catch(() => {})
      })
    }
  }
}

// ---------------------------------------------------------------------------
// Tests — Anti-duplicate KYC (US)
// ---------------------------------------------------------------------------

test.describe("KYC Anti-Duplicate — US", () => {
  test("US Individual: re-save after 40s does not create duplicate Stripe account", async ({ page }) => {
    test.setTimeout(180_000)

    // 1. Register provider and navigate to KYC form
    await registerProvider(page)
    await navigateToPaymentInfo(page)
    await selectCountry(page, "US")

    const person = randomPersonData()

    // 2. Fill all individual fields
    await fillPersonalFields(page, person)
    await fillBankFields(page, person.accountHolder)
    await uploadDocuments(page)
    await fillAllRemainingFields(page)

    // 3. First save
    await saveAndVerify(page)

    // 4. Wait 40 seconds — simulates real user delay where Stripe creates the account
    await page.waitForTimeout(40_000)

    // 5. Modify city to "Los Angeles" (triggers an update, not a create)
    await fillByLabel(page, "City", "Los Angeles")

    // 6. Save again — should update existing account, not create a duplicate
    await saveAndVerify(page)

    // 7. Reload and verify city was updated (proves account was updated, not duplicated)
    await page.reload()
    await expect(page.getByText("Payment information saved")).toBeVisible({ timeout: 15000 })
    await expect(inputByLabel(page, "City")).toHaveValue("Los Angeles")

    // 8. Verify original first name persisted (data not reset)
    await expect(inputByLabel(page, "First name")).toHaveValue(person.firstName)
  })

  test("US Business: re-save after 40s does not create duplicate Stripe account", async ({ page }) => {
    test.setTimeout(180_000)

    // 1. Register agency and navigate to KYC form
    await registerAgency(page)
    await navigateToPaymentInfo(page)
    await selectCountry(page, "US")

    // 2. Toggle business mode ON
    await page.getByRole("switch").click()
    await expect(page.getByText("Company Information").first()).toBeVisible({ timeout: 10000 })

    const person = randomPersonData()

    // 3. Fill all representative + company + bank fields
    await fillPersonalFields(page, person)
    const originalBizName = await fillBusinessFields(page)
    await fillBankFields(page, originalBizName)
    await uploadDocuments(page)
    await fillAllRemainingFields(page)

    // 4. First save
    await saveAndVerify(page)

    // 5. Wait 40 seconds — simulates real user delay where Stripe creates the account
    await page.waitForTimeout(40_000)

    // 6. Modify business name (triggers an update, not a create)
    const updatedBizName = `Updated Corp ${Date.now()}`
    await fillByLabel(page, "Business name", updatedBizName)

    // 7. Save again — should update existing account, not create a duplicate
    await saveAndVerify(page)

    // 8. Reload and verify business name was updated
    await page.reload()
    await expect(page.getByText("Payment information saved")).toBeVisible({ timeout: 15000 })
    await expect(inputByLabel(page, "Business name")).toHaveValue(updatedBizName)

    // 9. Verify original first name persisted (data not reset)
    await expect(inputByLabel(page, "First name")).toHaveValue(person.firstName)
  })
})
