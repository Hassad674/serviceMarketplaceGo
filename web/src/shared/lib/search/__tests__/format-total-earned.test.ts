import { describe, expect, it } from "vitest"
import {
  currencyForPricing,
  formatTotalEarned,
} from "../format-total-earned"

describe("formatTotalEarned", () => {
  it("returns an empty string for zero amounts", () => {
    expect(formatTotalEarned(0, "EUR", "fr")).toBe("")
    expect(formatTotalEarned(0, "EUR", "en")).toBe("")
  })

  it("returns an empty string for negative amounts", () => {
    expect(formatTotalEarned(-100, "EUR", "fr")).toBe("")
  })

  it("returns an empty string for NaN amounts", () => {
    expect(formatTotalEarned(Number.NaN, "EUR", "fr")).toBe("")
  })

  it("formats EUR in fr locale with trailing symbol", () => {
    const formatted = formatTotalEarned(1234500, "EUR", "fr")
    // Non-breaking spaces from Intl — we only assert the digits/currency
    // glyph are present to avoid whitespace fragility across Node versions.
    expect(formatted).toContain("12")
    expect(formatted).toContain("345")
    expect(formatted).toMatch(/€/)
  })

  it("formats EUR in en locale with leading symbol", () => {
    const formatted = formatTotalEarned(1234500, "EUR", "en")
    expect(formatted).toContain("12,345")
    expect(formatted).toMatch(/€/)
  })
})

describe("currencyForPricing", () => {
  it("returns EUR when pricing is null", () => {
    expect(currencyForPricing(null)).toBe("EUR")
    expect(currencyForPricing(undefined)).toBe("EUR")
  })

  it("returns EUR when pricing currency is the dimensionless pct", () => {
    expect(
      currencyForPricing({
        type: "commission_pct",
        min_amount: 1000,
        max_amount: 2000,
        currency: "pct",
        negotiable: false,
      }),
    ).toBe("EUR")
  })

  it("returns the pricing currency when set", () => {
    expect(
      currencyForPricing({
        type: "daily",
        min_amount: 60000,
        max_amount: null,
        currency: "USD",
        negotiable: false,
      }),
    ).toBe("USD")
  })
})
