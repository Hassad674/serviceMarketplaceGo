import path from "node:path"
import { test, expect, type Page } from "@playwright/test"
import { registerProvider, registerAgency } from "./helpers/auth"

type CountryTestData = {
  code: string
  name: string
  phone: string
  postalCode: string
  state?: string
  ssnLast4?: string
  idNumber?: string
  bankType: "iban" | "local"
  iban?: string
  bic?: string
  accountNumber?: string
  routingNumber?: string
  businessPostalCode: string
  businessState?: string
}

const COUNTRIES: Record<string, CountryTestData> = {
  US: {
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
  },
  DE: {
    code: "DE",
    name: "Germany",
    phone: "+4915112345678",
    postalCode: "10115",
    bankType: "iban",
    iban: "DE89370400440532013000",
    bic: "COBADEFFXXX",
    businessPostalCode: "80331",
  },
  GB: {
    code: "GB",
    name: "United Kingdom",
    phone: "+447911123456",
    postalCode: "SW1A 1AA",
    bankType: "iban",
    iban: "GB82WEST12345698765432",
    bic: "WESTGB2L",
    businessPostalCode: "EC2R 8AH",
  },
  SG: {
    code: "SG",
    name: "Singapore",
    phone: "+6591234567",
    postalCode: "018956",
    idNumber: "S1234567D",
    bankType: "local",
    accountNumber: "000123456789",
    routingNumber: "1100000",
    businessPostalCode: "049318",
  },
  IN: {
    code: "IN",
    name: "India",
    phone: "+919876543210",
    postalCode: "400001",
    state: "MH",
    idNumber: "ABCDE1234F",
    bankType: "local",
    accountNumber: "000123456789",
    routingNumber: "HDFC0000001",
    businessPostalCode: "110001",
    businessState: "DL",
  },
}

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

function randomPersonData(country: CountryTestData) {
  const firstName = pick(FIRST_NAMES)
  const lastName = pick(LAST_NAMES)
  const ts = Date.now()
  return {
    firstName,
    lastName,
    dob: "1990-05-15",
    email: `kyc-${country.code.toLowerCase()}-${ts}@test.com`,
    address: pick(STREETS),
    city: pick(CITIES),
    postalCode: country.postalCode,
    phone: country.phone,
    accountHolder: `${firstName} ${lastName}`,
  }
}

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
  // Wait for the form to re-render by checking that at least one input label appears
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
    // Dynamic form shows "Uploaded — pending verification" after successful upload
    await expect(
      page.getByText("pending verification").nth(i),
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

async function fillPersonalFields(page: Page, country: CountryTestData, person: ReturnType<typeof randomPersonData>) {
  await fillByLabel(page, "First name", person.firstName)
  await fillByLabel(page, "Last name", person.lastName)

  const dobLabel = page.locator("label", { hasText: "Date of birth" }).first()
  if (await dobLabel.isVisible().catch(() => false)) {
    const dobInput = dobLabel.locator("..").locator("input").first()
    await dobInput.scrollIntoViewIfNeeded()
    await dobInput.fill(person.dob)
  }

  // Email
  const emailLabel = page.locator("label", { hasText: "Email" }).first()
  if (await emailLabel.isVisible().catch(() => false)) {
    await fillByLabel(page, "Email", person.email)
  }

  await fillByLabel(page, "Address", person.address)
  await fillByLabel(page, "City", person.city)
  await fillByLabel(page, "Postal code", person.postalCode)
  await fillByLabel(page, "Phone number", person.phone)

  // State / Province — rendered as a <select> dropdown for known countries
  if (country.state) {
    const stateLabel = page.locator("label", { hasText: "State" }).first()
    if (await stateLabel.isVisible().catch(() => false)) {
      const stateSelect = stateLabel.locator("..").locator("select").first()
      if (await stateSelect.isVisible().catch(() => false)) {
        await stateSelect.scrollIntoViewIfNeeded()
        // Wait for options to load (lazy-loaded country-region-data)
        await page.waitForTimeout(1000)
        await stateSelect.selectOption(country.state)
      } else {
        // Fallback to text input if no select found
        const stateInput = stateLabel.locator("..").locator("input").first()
        await stateInput.scrollIntoViewIfNeeded()
        await stateInput.fill(country.state)
      }
    }
  }

  // SSN last 4 — US only
  if (country.ssnLast4) {
    const ssnLabel = page.locator("label", { hasText: "SSN" }).first()
    if (await ssnLabel.isVisible().catch(() => false)) {
      const ssnInput = ssnLabel.locator("..").locator("input").first()
      await ssnInput.scrollIntoViewIfNeeded()
      await ssnInput.fill(country.ssnLast4)
    }
  }

  // National ID Number — SG, IN, US
  if (country.idNumber) {
    const idLabel = page.locator("label", { hasText: "National ID" }).first()
    if (await idLabel.isVisible().catch(() => false)) {
      const idInput = idLabel.locator("..").locator("input").first()
      await idInput.scrollIntoViewIfNeeded()
      await idInput.fill(country.idNumber)
    }
  }

  // Political exposure — some countries show a select for this
  const peLabel = page.locator("label", { hasText: "Political Exposure" }).first()
  if (await peLabel.isVisible().catch(() => false)) {
    const peSelect = peLabel.locator("..").locator("select").first()
    await peSelect.scrollIntoViewIfNeeded()
    await peSelect.selectOption("none")
  }

  // Activity sector dropdown
  const sectorSelect = page.locator("select[aria-label='Activity sector']")
  if (await sectorSelect.isVisible().catch(() => false)) {
    await sectorSelect.selectOption({ index: 1 })
  }

  // Nationality select — pick the country itself
  const natLabel = page.locator("label", { hasText: "Nationality" }).first()
  if (await natLabel.isVisible().catch(() => false)) {
    const natSelect = natLabel.locator("..").locator("select").first()
    if (await natSelect.isVisible().catch(() => false)) {
      await natSelect.scrollIntoViewIfNeeded()
      await natSelect.selectOption(country.code)
    }
  }

  // Catch-all: fill any remaining empty text/email/tel inputs with generic data
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

  // Catch-all: select first non-empty option on any unselected selects
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

/**
 * Fill ALL remaining empty fields on the page — catch-all for exotic countries.
 * Runs after specific field helpers to pick up anything they missed.
 */
async function fillAllRemainingFields(page: Page, country: CountryTestData) {
  // Scroll through the entire page to make all elements "visible"
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

  // Fill ALL text/email/tel inputs on the page (not just visible)
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
        else if (type === "tel") await input.fill(country.phone).catch(() => {})
        else await input.fill("Test123").catch(() => {})
      }
    }
  }

  // Fill ALL empty selects
  const allSelects = page.locator("select")
  const selectCount = await allSelects.count()
  for (let i = 0; i < selectCount; i++) {
    const sel = allSelects.nth(i)
    const val = await sel.inputValue().catch(() => "SKIP")
    if (val === "") {
      await sel.scrollIntoViewIfNeeded().catch(() => {})
      await sel.selectOption(country.code).catch(async () => {
        await sel.selectOption({ index: 1 }).catch(() => {})
      })
    }
  }
}

async function fillBankFields(page: Page, country: CountryTestData, accountHolder: string) {
  if (country.bankType === "iban") {
    await fillByLabel(page, "IBAN", country.iban!)
    const bicLabel = page.locator("label", { hasText: "BIC" }).first()
    if (await bicLabel.isVisible().catch(() => false)) {
      const bicInput = bicLabel.locator("..").locator("input").first()
      await bicInput.scrollIntoViewIfNeeded()
      await bicInput.fill(country.bic ?? "")
    }
  } else {
    await fillByLabel(page, "Account number", country.accountNumber!)
    await fillByLabel(page, "Routing number", country.routingNumber!)
  }

  await fillByLabel(page, "Account holder name", accountHolder)

  const bankCountryLabel = page.locator("label", { hasText: "Bank country" }).first()
  if (await bankCountryLabel.isVisible().catch(() => false)) {
    const bankCountrySelect = bankCountryLabel.locator("..").locator("select").first()
    await bankCountrySelect.scrollIntoViewIfNeeded()
    await bankCountrySelect.selectOption(country.code)
  }
}

async function fillBusinessFields(page: Page, country: CountryTestData) {
  const ts = Date.now()
  const bizName = `Test Corp ${ts}`

  // Business role dropdown
  const roleSelect = page.locator("select[aria-label='Your role in the company']")
  if (await roleSelect.isVisible().catch(() => false)) {
    await roleSelect.scrollIntoViewIfNeeded()
    await roleSelect.selectOption("ceo")
  }

  // Representative Title / Role
  const titleLabel = page.locator("label", { hasText: "Title / Role" }).first()
  if (await titleLabel.isVisible().catch(() => false)) {
    await fillByLabel(page, "Title / Role", "CEO")
  }

  // Company info
  await fillByLabel(page, "Business name", bizName)
  await fillByLabel(page, "Business address", pick(STREETS))
  await fillByLabel(page, "Business city", pick(CITIES))
  await fillByLabel(page, "Business postal code", country.businessPostalCode)

  // Business state — US, IN — rendered as a <select> dropdown
  if (country.businessState) {
    const stateLabel = page.locator("label", { hasText: "Business state" }).first()
    if (await stateLabel.isVisible().catch(() => false)) {
      const stateSelect = stateLabel.locator("..").locator("select").first()
      if (await stateSelect.isVisible().catch(() => false)) {
        await stateSelect.scrollIntoViewIfNeeded()
        // Wait for options to load (lazy-loaded country-region-data)
        await page.waitForTimeout(1000)
        await stateSelect.selectOption(country.businessState)
      } else {
        const stateInput = stateLabel.locator("..").locator("input").first()
        await stateInput.scrollIntoViewIfNeeded()
        await stateInput.fill(country.businessState)
      }
    }
  }

  // Tax ID
  const taxLabel = page.locator("label", { hasText: "Tax ID" }).first()
  if (await taxLabel.isVisible().catch(() => false)) {
    const taxIdValue = String(ts).slice(-14).padStart(14, "0")
    await fillByLabel(page, "Tax ID", taxIdValue)
  }

  // Company phone (second Phone number on page)
  const phoneLabels = page.locator("label", { hasText: "Phone number" })
  const phoneCount = await phoneLabels.count()
  if (phoneCount > 1) {
    const companyPhoneInput = phoneLabels.nth(1).locator("..").locator("input").first()
    if (await companyPhoneInput.isVisible().catch(() => false)) {
      await companyPhoneInput.scrollIntoViewIfNeeded()
      await companyPhoneInput.fill(country.phone)
    }
  }

  // Catch-all for business sections: fill any empty inputs
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

  // Catch-all: select first option on any empty selects
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

test.describe("KYC Flow — US", () => {
  test("US Individual: register, fill fields, upload docs, save", async ({ page }) => {
    test.setTimeout(120_000)
    await registerProvider(page)
    await navigateToPaymentInfo(page)
    await selectCountry(page, "US")

    const country = COUNTRIES.US
    const person = randomPersonData(country)

    await fillPersonalFields(page, country, person)
    await fillBankFields(page, country, person.accountHolder)
    await uploadDocuments(page)
    await saveAndVerify(page)

    // Verify persistence
    await page.reload()
    await expect(page.getByText("Payment information saved")).toBeVisible({ timeout: 15000 })
    await expect(inputByLabel(page, "First name")).toHaveValue(person.firstName)
    await expect(inputByLabel(page, "Last name")).toHaveValue(person.lastName)
    await expect(inputByLabel(page, "Account holder name")).toHaveValue(person.accountHolder)
  })

  test("US Business: register, toggle business, fill fields, save", async ({ page }) => {
    test.setTimeout(120_000)
    await registerAgency(page)
    await navigateToPaymentInfo(page)
    await selectCountry(page, "US")

    await page.getByRole("switch").click()
    await expect(page.getByText("Company Information").first()).toBeVisible({ timeout: 10000 })

    const country = COUNTRIES.US
    const person = randomPersonData(country)

    await fillPersonalFields(page, country, person)
    const bizName = await fillBusinessFields(page, country)
    await fillBankFields(page, country, bizName)
    await uploadDocuments(page)
    await saveAndVerify(page)

    await page.reload()
    await expect(page.getByText("Payment information saved")).toBeVisible({ timeout: 15000 })
    await expect(inputByLabel(page, "First name")).toHaveValue(person.firstName)
    await expect(inputByLabel(page, "Business name")).toHaveValue(bizName)
  })
})

test.describe("KYC Flow — DE", () => {
  test("DE Individual: register, fill fields, upload docs, save", async ({ page }) => {
    test.setTimeout(120_000)
    await registerProvider(page)
    await navigateToPaymentInfo(page)
    await selectCountry(page, "DE")

    const country = COUNTRIES.DE
    const person = randomPersonData(country)

    await fillPersonalFields(page, country, person)
    await fillBankFields(page, country, person.accountHolder)
    await uploadDocuments(page)
    await saveAndVerify(page)

    await page.reload()
    await expect(page.getByText("Payment information saved")).toBeVisible({ timeout: 15000 })
    await expect(inputByLabel(page, "First name")).toHaveValue(person.firstName)
    await expect(inputByLabel(page, "IBAN")).toHaveValue(country.iban!)
  })

  test("DE Business: register, toggle business, fill fields, save", async ({ page }) => {
    test.setTimeout(120_000)
    await registerAgency(page)
    await navigateToPaymentInfo(page)
    await selectCountry(page, "DE")

    await page.getByRole("switch").click()
    await expect(page.getByText("Company Information").first()).toBeVisible({ timeout: 10000 })

    const country = COUNTRIES.DE
    const person = randomPersonData(country)

    await fillPersonalFields(page, country, person)
    const bizName = await fillBusinessFields(page, country)
    await fillBankFields(page, country, bizName)
    await uploadDocuments(page)
    await saveAndVerify(page)

    await page.reload()
    await expect(page.getByText("Payment information saved")).toBeVisible({ timeout: 15000 })
    await expect(inputByLabel(page, "IBAN")).toHaveValue(country.iban!)
    await expect(inputByLabel(page, "Business name")).toHaveValue(bizName)
  })
})

test.describe("KYC Flow — GB", () => {
  test("GB Individual: register, fill fields, upload docs, save", async ({ page }) => {
    test.setTimeout(120_000)
    await registerProvider(page)
    await navigateToPaymentInfo(page)
    await selectCountry(page, "GB")

    const country = COUNTRIES.GB
    const person = randomPersonData(country)

    await fillPersonalFields(page, country, person)
    await fillBankFields(page, country, person.accountHolder)
    await uploadDocuments(page)
    await saveAndVerify(page)

    await page.reload()
    await expect(page.getByText("Payment information saved")).toBeVisible({ timeout: 15000 })
    await expect(inputByLabel(page, "First name")).toHaveValue(person.firstName)
    await expect(inputByLabel(page, "IBAN")).toHaveValue(country.iban!)
  })

  test("GB Business: register, toggle business, fill fields, save", async ({ page }) => {
    test.setTimeout(120_000)
    await registerAgency(page)
    await navigateToPaymentInfo(page)
    await selectCountry(page, "GB")

    await page.getByRole("switch").click()
    await expect(page.getByText("Company Information").first()).toBeVisible({ timeout: 10000 })

    const country = COUNTRIES.GB
    const person = randomPersonData(country)

    await fillPersonalFields(page, country, person)
    const bizName = await fillBusinessFields(page, country)
    await fillBankFields(page, country, bizName)
    await uploadDocuments(page)
    await saveAndVerify(page)

    await page.reload()
    await expect(page.getByText("Payment information saved")).toBeVisible({ timeout: 15000 })
    await expect(inputByLabel(page, "IBAN")).toHaveValue(country.iban!)
    await expect(inputByLabel(page, "Business name")).toHaveValue(bizName)
  })
})

test.describe("KYC Flow — SG", () => {
  test("SG Individual: register, fill fields, upload docs, save", async ({ page }) => {
    test.setTimeout(120_000)
    await registerProvider(page)
    await navigateToPaymentInfo(page)
    await selectCountry(page, "SG")

    const country = COUNTRIES.SG
    const person = randomPersonData(country)

    await fillPersonalFields(page, country, person)
    await fillBankFields(page, country, person.accountHolder)
    await fillAllRemainingFields(page, country)
    await uploadDocuments(page)
    await saveAndVerify(page)

    await page.reload()
    await expect(page.getByText("Payment information saved")).toBeVisible({ timeout: 15000 })
    await expect(inputByLabel(page, "First name")).toHaveValue(person.firstName)
    await expect(inputByLabel(page, "Account holder name")).toHaveValue(person.accountHolder)
  })

  test("SG Business: register, toggle business, fill fields, save", async ({ page }) => {
    test.setTimeout(120_000)
    await registerAgency(page)
    await navigateToPaymentInfo(page)
    await selectCountry(page, "SG")

    await page.getByRole("switch").click()
    await expect(page.getByText("Company Information").first()).toBeVisible({ timeout: 10000 })

    const country = COUNTRIES.SG
    const person = randomPersonData(country)

    await fillPersonalFields(page, country, person)
    const bizName = await fillBusinessFields(page, country)
    await fillBankFields(page, country, bizName)
    await fillAllRemainingFields(page, country)
    await uploadDocuments(page)
    await saveAndVerify(page)

    await page.reload()
    await expect(page.getByText("Payment information saved")).toBeVisible({ timeout: 15000 })
    await expect(inputByLabel(page, "First name")).toHaveValue(person.firstName)
    await expect(inputByLabel(page, "Business name")).toHaveValue(bizName)
  })
})

test.describe("KYC Flow — IN", () => {
  test("IN Individual: register, fill fields, upload docs, save", async ({ page }) => {
    test.setTimeout(120_000)
    await registerProvider(page)
    await navigateToPaymentInfo(page)
    await selectCountry(page, "IN")

    const country = COUNTRIES.IN
    const person = randomPersonData(country)

    await fillPersonalFields(page, country, person)
    await fillBankFields(page, country, person.accountHolder)
    await uploadDocuments(page)
    await saveAndVerify(page)

    await page.reload()
    await expect(page.getByText("Payment information saved")).toBeVisible({ timeout: 15000 })
    await expect(inputByLabel(page, "First name")).toHaveValue(person.firstName)
    await expect(inputByLabel(page, "Account holder name")).toHaveValue(person.accountHolder)
  })

  test("IN Business: register, toggle business, fill fields, save", async ({ page }) => {
    test.setTimeout(120_000)
    await registerAgency(page)
    await navigateToPaymentInfo(page)
    await selectCountry(page, "IN")

    await page.getByRole("switch").click()
    await expect(page.getByText("Company Information").first()).toBeVisible({ timeout: 10000 })

    const country = COUNTRIES.IN
    const person = randomPersonData(country)

    await fillPersonalFields(page, country, person)
    const bizName = await fillBusinessFields(page, country)
    await fillBankFields(page, country, bizName)
    await fillAllRemainingFields(page, country)
    await uploadDocuments(page)
    await saveAndVerify(page)

    await page.reload()
    await expect(page.getByText("Payment information saved")).toBeVisible({ timeout: 15000 })
    await expect(inputByLabel(page, "First name")).toHaveValue(person.firstName)
    await expect(inputByLabel(page, "Business name")).toHaveValue(bizName)
  })
})
