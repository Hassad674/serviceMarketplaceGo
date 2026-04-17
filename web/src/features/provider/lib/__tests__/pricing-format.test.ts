import { describe, expect, it } from "vitest"
import { formatPricing } from "../pricing-format"
import type { Pricing } from "../../api/profile-api"

function row(overrides: Partial<Pricing>): Pricing {
  return {
    kind: "direct",
    type: "daily",
    min_amount: 50000,
    max_amount: null,
    currency: "EUR",
    note: "",
    negotiable: false,
    ...overrides,
  }
}

describe("formatPricing", () => {
  it("formats daily EUR in French", () => {
    const result = formatPricing(
      row({ type: "daily", min_amount: 50000, currency: "EUR" }),
      "fr",
    )
    expect(result).toContain("500")
    expect(result).toContain("€")
    expect(result).toContain("/j")
  })

  it("formats daily EUR in English", () => {
    const result = formatPricing(
      row({ type: "daily", min_amount: 50000, currency: "EUR" }),
      "en",
    )
    expect(result).toContain("500")
    expect(result).toContain("€")
    expect(result).toContain("/day")
  })

  it("formats hourly USD in English", () => {
    const result = formatPricing(
      row({ type: "hourly", min_amount: 7500, currency: "USD" }),
      "en",
    )
    expect(result).toContain("75")
    expect(result).toContain("$")
    expect(result).toContain("/hr")
  })

  it("formats project_from EUR with FR prefix", () => {
    const result = formatPricing(
      row({ type: "project_from", min_amount: 1000000, currency: "EUR" }),
      "fr",
    )
    expect(result).toMatch(/À partir de/)
    expect(result).toContain("10")
    expect(result).toContain("000")
    expect(result).toContain("€")
  })

  it("formats project_from EUR with EN prefix", () => {
    const result = formatPricing(
      row({ type: "project_from", min_amount: 1000000, currency: "EUR" }),
      "en",
    )
    expect(result).toMatch(/From/)
  })

  it("formats project_range EUR with both ends and single currency symbol on the max side", () => {
    const result = formatPricing(
      row({
        type: "project_range",
        min_amount: 1500000,
        max_amount: 5000000,
        currency: "EUR",
      }),
      "fr",
    )
    expect(result).toMatch(/15[\s\u202f\u00a0]?000\s?–\s?50[\s\u202f\u00a0]?000\s?€/)
  })

  it("formats commission_pct with trailing %", () => {
    const result = formatPricing(
      row({
        type: "commission_pct",
        min_amount: 500,
        max_amount: 1500,
        currency: "pct",
      }),
      "fr",
    )
    expect(result).toContain("5")
    expect(result).toContain("15")
    expect(result).toContain("%")
  })

  it("formats commission_pct without max as a single percent", () => {
    const result = formatPricing(
      row({
        type: "commission_pct",
        min_amount: 550,
        max_amount: null,
        currency: "pct",
      }),
      "fr",
    )
    expect(result).toContain("5,5")
    expect(result).toContain("%")
    expect(result).not.toContain("–")
  })

  // V1 pricing simplification: the referrer editor stores the single
  // user-entered percentage in BOTH min and max. formatPricing must
  // collapse the range when the two bounds are equal so the card
  // surfaces "10 % de commission" instead of "10 – 10 %".
  it("collapses commission_pct to a single headline when min equals max (V1)", () => {
    const resultFR = formatPricing(
      row({
        type: "commission_pct",
        min_amount: 1000,
        max_amount: 1000,
        currency: "pct",
      }),
      "fr",
    )
    expect(resultFR).toContain("10")
    expect(resultFR).toContain("%")
    expect(resultFR).toContain("commission")
    expect(resultFR).not.toContain("–")

    const resultEN = formatPricing(
      row({
        type: "commission_pct",
        min_amount: 1000,
        max_amount: 1000,
        currency: "pct",
      }),
      "en",
    )
    expect(resultEN).toContain("10")
    expect(resultEN).toContain("%")
    expect(resultEN).toContain("commission")
    expect(resultEN).not.toContain("–")
  })

  it("formats commission_flat EUR with 'per deal' suffix in English", () => {
    const result = formatPricing(
      row({
        type: "commission_flat",
        min_amount: 300000,
        currency: "EUR",
      }),
      "en",
    )
    expect(result).toContain("3,000")
    expect(result).toMatch(/per deal/)
  })

  it("formats commission_flat EUR with '/ deal' suffix in French", () => {
    const result = formatPricing(
      row({
        type: "commission_flat",
        min_amount: 300000,
        currency: "EUR",
      }),
      "fr",
    )
    expect(result).toMatch(/deal/)
  })

  it("handles GBP hourly rates in English", () => {
    const result = formatPricing(
      row({ type: "hourly", min_amount: 10000, currency: "GBP" }),
      "en",
    )
    expect(result).toContain("£")
    expect(result).toContain("100")
  })

  it("handles CAD daily rates in French", () => {
    const result = formatPricing(
      row({ type: "daily", min_amount: 80000, currency: "CAD" }),
      "fr",
    )
    expect(result).toContain("800")
    expect(result).toContain("/j")
  })
})
