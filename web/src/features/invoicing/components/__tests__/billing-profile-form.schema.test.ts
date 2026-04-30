import { describe, it, expect } from "vitest"
import {
  billingProfileFormSchema,
  type BillingProfileFormValues,
} from "../billing-profile-form.schema"

function validProfile(
  overrides: Partial<BillingProfileFormValues> = {},
): BillingProfileFormValues {
  return {
    profile_type: "business",
    legal_name: "Acme SAS",
    trading_name: "Acme",
    legal_form: "SAS",
    tax_id: "12345678901234",
    vat_number: "FR12345678901",
    address_line1: "1 rue de la Paix",
    address_line2: "",
    postal_code: "75001",
    city: "Paris",
    country: "FR",
    invoicing_email: "billing@acme.com",
    ...overrides,
  }
}

describe("billingProfileFormSchema — happy paths", () => {
  it("accepts a fully valid French SAS", () => {
    const result = billingProfileFormSchema.safeParse(validProfile())
    expect(result.success).toBe(true)
  })

  it("accepts a non-EU country with no SIRET / no VAT", () => {
    const result = billingProfileFormSchema.safeParse(
      validProfile({
        country: "US",
        tax_id: "",
        vat_number: "",
      }),
    )
    expect(result.success).toBe(true)
  })

  it("accepts an EU non-FR country with empty VAT number (still allowed)", () => {
    const result = billingProfileFormSchema.safeParse(
      validProfile({
        country: "BE",
        tax_id: "",
        vat_number: "",
      }),
    )
    expect(result.success).toBe(true)
  })

  it("accepts an EU non-FR country with a well-formed VAT number", () => {
    const result = billingProfileFormSchema.safeParse(
      validProfile({
        country: "BE",
        tax_id: "",
        vat_number: "BE0123456789",
      }),
    )
    expect(result.success).toBe(true)
  })

  it("accepts an individual profile with no legal_form", () => {
    const result = billingProfileFormSchema.safeParse(
      validProfile({ profile_type: "individual", legal_form: "" }),
    )
    expect(result.success).toBe(true)
  })
})

describe("billingProfileFormSchema — required fields", () => {
  it.each([
    ["legal_name", ""],
    ["address_line1", ""],
    ["postal_code", ""],
    ["city", ""],
    ["country", ""],
  ] as const)("rejects empty %s", (field, value) => {
    const result = billingProfileFormSchema.safeParse(
      validProfile({ [field]: value } as Partial<BillingProfileFormValues>),
    )
    expect(result.success).toBe(false)
    if (!result.success) {
      expect(result.error.issues.some((i) => i.path[0] === field)).toBe(true)
    }
  })

  it("rejects an invalid country (length != 2)", () => {
    const result = billingProfileFormSchema.safeParse(
      validProfile({ country: "FRA" }),
    )
    expect(result.success).toBe(false)
  })
})

describe("billingProfileFormSchema — FR SIRET rule", () => {
  it("rejects FR with non-14-digit SIRET", () => {
    const result = billingProfileFormSchema.safeParse(
      validProfile({ tax_id: "1234567890" }),
    )
    expect(result.success).toBe(false)
    if (!result.success) {
      expect(result.error.issues.some((i) => i.path[0] === "tax_id")).toBe(true)
    }
  })

  it("rejects FR with SIRET containing non-digits", () => {
    const result = billingProfileFormSchema.safeParse(
      validProfile({ tax_id: "ABCDEFGHIJKLMN" }),
    )
    expect(result.success).toBe(false)
  })

  it("strips internal spaces before SIRET length check", () => {
    const result = billingProfileFormSchema.safeParse(
      validProfile({ tax_id: "12345 67890 1234" }),
    )
    expect(result.success).toBe(true)
  })
})

describe("billingProfileFormSchema — business legal_form rule", () => {
  it("rejects business profile with empty legal_form", () => {
    const result = billingProfileFormSchema.safeParse(
      validProfile({ profile_type: "business", legal_form: "" }),
    )
    expect(result.success).toBe(false)
    if (!result.success) {
      expect(result.error.issues.some((i) => i.path[0] === "legal_form")).toBe(
        true,
      )
    }
  })

  it("accepts business profile with a legal_form filled in", () => {
    const result = billingProfileFormSchema.safeParse(
      validProfile({ profile_type: "business", legal_form: "EURL" }),
    )
    expect(result.success).toBe(true)
  })
})

describe("billingProfileFormSchema — EU non-FR VAT rule", () => {
  it("rejects malformed VAT number for an EU country", () => {
    const result = billingProfileFormSchema.safeParse(
      validProfile({
        country: "BE",
        tax_id: "",
        vat_number: "not-a-vat",
      }),
    )
    expect(result.success).toBe(false)
    if (!result.success) {
      expect(result.error.issues.some((i) => i.path[0] === "vat_number")).toBe(
        true,
      )
    }
  })

  it("does NOT enforce VAT format for non-EU countries", () => {
    const result = billingProfileFormSchema.safeParse(
      validProfile({
        country: "US",
        tax_id: "",
        vat_number: "garbage",
      }),
    )
    // US: VAT rule is skipped — saving a "garbage" VAT is allowed
    // because the EU branch only kicks in for EU codes.
    expect(result.success).toBe(true)
  })

  it("does NOT enforce VAT format for FR (SIRET takes the slot)", () => {
    const result = billingProfileFormSchema.safeParse(
      validProfile({ vat_number: "garbage" }),
    )
    // FR keeps SIRET valid; the VAT rule is skipped for FR.
    expect(result.success).toBe(true)
  })

  it("accepts uppercase VAT numbers with spaces", () => {
    const result = billingProfileFormSchema.safeParse(
      validProfile({
        country: "BE",
        tax_id: "",
        vat_number: "BE 0123 456 789",
      }),
    )
    expect(result.success).toBe(true)
  })
})
