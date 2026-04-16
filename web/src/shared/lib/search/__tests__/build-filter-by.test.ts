/**
 * build-filter-by.test.ts is the parity test for the TS filter
 * builder. The same scenarios are tested in the Go side
 * (`backend/internal/app/search/filter_builder_test.go`); both
 * tests must produce identical filter_by strings.
 */

import { describe, expect, it } from "vitest"
import { buildFilterBy, type SearchFilterInput } from "../build-filter-by"

describe("buildFilterBy", () => {
  it("returns empty string for empty input", () => {
    expect(buildFilterBy({})).toBe("")
  })

  it("emits availability_status clause for non-empty list", () => {
    expect(buildFilterBy({ availabilityStatus: ["available_now"] })).toBe(
      "availability_status:[available_now]",
    )
    expect(
      buildFilterBy({ availabilityStatus: ["available_now", "available_soon"] }),
    ).toBe("availability_status:[available_now,available_soon]")
  })

  it("dedupes and trims string slices", () => {
    expect(
      buildFilterBy({
        availabilityStatus: [" available_now ", "available_now", "  "],
      }),
    ).toBe("availability_status:[available_now]")
  })

  it("emits pricing range clause", () => {
    expect(buildFilterBy({ pricingMin: 50000, pricingMax: 150000 })).toBe(
      "pricing_min_amount:>=50000 && pricing_min_amount:<=150000",
    )
    expect(buildFilterBy({ pricingMin: 50000 })).toBe("pricing_min_amount:>=50000")
    expect(buildFilterBy({ pricingMax: 150000 })).toBe("pricing_min_amount:<=150000")
  })

  it("emits case-insensitive city clause with backticks", () => {
    expect(buildFilterBy({ city: "Paris" })).toBe("city:`Paris`")
    expect(buildFilterBy({ city: "New York" })).toBe("city:`New York`")
    expect(buildFilterBy({ city: "  Lyon  " })).toBe("city:`Lyon`")
    expect(buildFilterBy({ city: "" })).toBe("")
    expect(buildFilterBy({ city: "   " })).toBe("")
  })

  it("preserves country code casing (match is case-insensitive)", () => {
    expect(buildFilterBy({ countryCode: "FR" })).toBe("country_code:FR")
    expect(buildFilterBy({ countryCode: "fr" })).toBe("country_code:fr")
    expect(buildFilterBy({ countryCode: "" })).toBe("")
  })

  it("emits geo clause when lat/lng/radius all set", () => {
    expect(
      buildFilterBy({ geoLat: 48.8566, geoLng: 2.3522, geoRadiusKm: 25 }),
    ).toBe("location:(48.8566,2.3522,25 km)")
  })

  it("drops geo clause when any field is missing", () => {
    expect(buildFilterBy({ geoLat: 48.8566 })).toBe("")
    expect(buildFilterBy({ geoLat: 48.8566, geoLng: 2.3522 })).toBe("")
    expect(
      buildFilterBy({ geoLat: 48.8566, geoLng: 2.3522, geoRadiusKm: 0 }),
    ).toBe("")
  })

  it("emits language / expertise / skill list clauses", () => {
    expect(buildFilterBy({ languages: ["fr", "en"] })).toBe(
      "languages_professional:[fr,en]",
    )
    expect(buildFilterBy({ expertiseDomains: ["dev", "design"] })).toBe(
      "expertise_domains:[dev,design]",
    )
    expect(buildFilterBy({ skills: ["react", "go"] })).toBe("skills:[react,go]")
  })

  it("emits rating clause with smart number formatting", () => {
    expect(buildFilterBy({ ratingMin: 4 })).toBe("rating_average:>=4")
    expect(buildFilterBy({ ratingMin: 4.5 })).toBe("rating_average:>=4.5")
    expect(buildFilterBy({ ratingMin: 0 })).toBe("")
    expect(buildFilterBy({ ratingMin: -1 })).toBe("")
  })

  it("emits work_mode clause", () => {
    expect(buildFilterBy({ workMode: ["remote", "hybrid"] })).toBe(
      "work_mode:[remote,hybrid]",
    )
  })

  it("emits boolean toggles", () => {
    expect(buildFilterBy({ isVerified: true })).toBe("is_verified:=true")
    expect(buildFilterBy({ isVerified: false })).toBe("is_verified:=false")
    expect(buildFilterBy({ isTopRated: true })).toBe("is_top_rated:=true")
    expect(buildFilterBy({ negotiable: true })).toBe("pricing_negotiable:=true")
  })

  it("preserves stable order across all clauses (parity with Go)", () => {
    const got = buildFilterBy({
      availabilityStatus: ["available_now"],
      pricingMin: 40000,
      pricingMax: 120000,
      city: "Paris",
      countryCode: "FR",
      languages: ["fr", "en"],
      skills: ["react"],
      ratingMin: 4,
      workMode: ["remote"],
      isVerified: true,
      isTopRated: true,
    })
    expect(got).toBe(
      "availability_status:[available_now]" +
        " && pricing_min_amount:>=40000 && pricing_min_amount:<=120000" +
        " && city:`Paris`" +
        " && country_code:FR" +
        " && languages_professional:[fr,en]" +
        " && skills:[react]" +
        " && rating_average:>=4" +
        " && work_mode:[remote]" +
        " && is_verified:=true" +
        " && is_top_rated:=true",
    )
  })

  it("is deterministic for repeated calls", () => {
    const input: SearchFilterInput = {
      languages: ["fr"],
      skills: ["go", "react"],
    }
    const a = buildFilterBy(input)
    const b = buildFilterBy(input)
    expect(a).toBe(b)
  })
})
