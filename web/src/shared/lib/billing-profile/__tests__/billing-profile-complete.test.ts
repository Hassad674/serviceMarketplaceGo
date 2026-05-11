// Unit tests for `checkBillingProfileComplete` — the pure client-side
// completeness mirror of the backend rule. The backend's `is_complete`
// flag remains the source of truth, but this helper is used by:
//   - The embedded billing card on the payment page to decide whether
//     to render the read-only summary or the full form on first paint.
//   - Tests that need a deterministic completeness check without a
//     live backend.
//
// Table-driven coverage: every required field, the business-only
// tax_id rule, edge cases (whitespace-only, undefined, partial input).

import { describe, expect, it } from "vitest"
import { checkBillingProfileComplete } from "../billing-profile-complete"

const fullIndividual = {
  profile_type: "individual" as const,
  legal_name: "Alice Martin",
  country: "FR",
  address_line1: "12 rue de la Paix",
  postal_code: "75001",
  city: "Paris",
  tax_id: "",
}

const fullBusiness = {
  profile_type: "business" as const,
  legal_name: "Acme Studio SARL",
  country: "FR",
  address_line1: "12 rue de la Paix",
  postal_code: "75001",
  city: "Paris",
  tax_id: "12345678901234",
}

describe("checkBillingProfileComplete", () => {
  it("returns false for null", () => {
    expect(checkBillingProfileComplete(null)).toBe(false)
  })

  it("returns false for undefined", () => {
    expect(checkBillingProfileComplete(undefined)).toBe(false)
  })

  it("returns false for an empty object", () => {
    expect(checkBillingProfileComplete({})).toBe(false)
  })

  it("returns true for a fully-filled individual profile (no tax_id required)", () => {
    expect(checkBillingProfileComplete(fullIndividual)).toBe(true)
  })

  it("returns true for a fully-filled business profile with a tax_id", () => {
    expect(checkBillingProfileComplete(fullBusiness)).toBe(true)
  })

  it("returns false when a business profile misses tax_id", () => {
    expect(
      checkBillingProfileComplete({ ...fullBusiness, tax_id: "" }),
    ).toBe(false)
  })

  it("returns false when a required string is whitespace-only", () => {
    expect(
      checkBillingProfileComplete({ ...fullIndividual, city: "   " }),
    ).toBe(false)
  })

  it("returns false when legal_name is missing", () => {
    expect(
      checkBillingProfileComplete({ ...fullIndividual, legal_name: "" }),
    ).toBe(false)
  })

  it("returns false when country is missing", () => {
    expect(
      checkBillingProfileComplete({ ...fullIndividual, country: "" }),
    ).toBe(false)
  })

  it("returns false when address_line1 is missing", () => {
    expect(
      checkBillingProfileComplete({
        ...fullIndividual,
        address_line1: "",
      }),
    ).toBe(false)
  })

  it("returns false when postal_code is missing", () => {
    expect(
      checkBillingProfileComplete({
        ...fullIndividual,
        postal_code: "",
      }),
    ).toBe(false)
  })

  it("returns false when city is missing", () => {
    expect(
      checkBillingProfileComplete({ ...fullIndividual, city: "" }),
    ).toBe(false)
  })

  it("ignores optional fields (vat_number, trading_name, address_line2, invoicing_email)", () => {
    // None of these appear in the input — completeness still passes.
    expect(checkBillingProfileComplete(fullIndividual)).toBe(true)
  })

  it("returns false when a business profile has tax_id as whitespace only", () => {
    expect(
      checkBillingProfileComplete({ ...fullBusiness, tax_id: "  " }),
    ).toBe(false)
  })

  it("returns true when an individual profile has empty tax_id", () => {
    // Individuals never need tax_id — the helper must NOT reject them.
    expect(
      checkBillingProfileComplete({ ...fullIndividual, tax_id: "" }),
    ).toBe(true)
  })

  it("returns false when profile_type is undefined and tax_id is missing", () => {
    // Without an explicit profile_type the helper treats the row as
    // non-business (lenient) — but the other required fields still
    // gate the result.
    const partial = {
      legal_name: "X",
      country: "FR",
      address_line1: "a",
      postal_code: "12345",
      city: "",
    }
    expect(checkBillingProfileComplete(partial)).toBe(false)
  })
})
