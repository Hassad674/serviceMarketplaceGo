/**
 * typesense-document-adapter.test.ts pins the conversion from the
 * raw Typesense document shape into the frozen UI SearchDocument
 * shape consumed by the result card. Defaults must match the
 * legacy SQL adapter so swapping data sources is transparent.
 */

import { describe, expect, it } from "vitest"
import type { RawSearchDocument } from "../typesense-client"
import { fromTypesenseDocument } from "../typesense-document-adapter"

const baseRaw: RawSearchDocument = {
  id: "11111111-1111-1111-1111-111111111111",
  persona: "freelance",
  is_published: true,
  display_name: "Alice",
  title: "Senior Go Developer",
  photo_url: "https://example.com/alice.jpg",
  city: "Paris",
  country_code: "fr",
  location: [48.8566, 2.3522],
  work_mode: ["remote"],
  languages_professional: ["fr", "en"],
  languages_conversational: [],
  availability_status: "available_now",
  availability_priority: 3,
  expertise_domains: ["dev"],
  skills: ["go", "react", "kubernetes", "aws", "postgresql", "redis", "extra-skill"],
  skills_text: "go react kubernetes aws postgresql redis extra-skill",
  pricing_type: "daily",
  pricing_min_amount: 60000,
  pricing_max_amount: 80000,
  pricing_currency: "EUR",
  pricing_negotiable: false,
  rating_average: 4.8,
  rating_count: 12,
  rating_score: 12.5,
  total_earned: 12345,
  completed_projects: 8,
  profile_completion_score: 90,
  last_active_at: 1700000000,
  response_rate: 0.95,
  is_verified: true,
  is_top_rated: true,
  is_featured: false,
  created_at: 1700000000,
  updated_at: 1700000100,
}

describe("fromTypesenseDocument", () => {
  it("maps the canonical happy path", () => {
    const got = fromTypesenseDocument(baseRaw)
    expect(got.id).toBe(baseRaw.id)
    expect(got.persona).toBe("freelance")
    expect(got.display_name).toBe("Alice")
    expect(got.title).toBe("Senior Go Developer")
    expect(got.city).toBe("Paris")
    expect(got.languages_professional).toEqual(["fr", "en"])
    expect(got.expertise_domains).toEqual(["dev"])
    expect(got.rating).toEqual({ average: 4.8, count: 12 })
    expect(got.total_earned).toBe(12345)
    expect(got.completed_projects).toBe(8)
  })

  it("caps skills at six entries", () => {
    const got = fromTypesenseDocument(baseRaw)
    expect(got.skills).toHaveLength(6)
    expect(got.skills).toEqual([
      "go",
      "react",
      "kubernetes",
      "aws",
      "postgresql",
      "redis",
    ])
  })

  it("converts created_at unix epoch to ISO string", () => {
    const got = fromTypesenseDocument(baseRaw)
    expect(got.created_at).toBe(new Date(1700000000 * 1000).toISOString())
  })

  it("returns null pricing when pricing_type missing", () => {
    const raw = { ...baseRaw, pricing_type: undefined }
    expect(fromTypesenseDocument(raw).pricing).toBeNull()
  })

  it("returns null pricing for unknown type", () => {
    const raw = { ...baseRaw, pricing_type: "unknown_type" }
    expect(fromTypesenseDocument(raw).pricing).toBeNull()
  })

  it("returns valid pricing object", () => {
    const got = fromTypesenseDocument(baseRaw)
    expect(got.pricing).toEqual({
      type: "daily",
      min_amount: 60000,
      max_amount: 80000,
      currency: "EUR",
      negotiable: false,
    })
  })

  it("falls back to default availability when value is unrecognised", () => {
    const raw = { ...baseRaw, availability_status: "weird" }
    expect(fromTypesenseDocument(raw).availability_status).toBe("available_now")
  })

  it("tolerates missing optional fields", () => {
    const minimal: RawSearchDocument = {
      ...baseRaw,
      title: undefined,
      photo_url: undefined,
      city: undefined,
      country_code: undefined,
      pricing_type: undefined,
    }
    const got = fromTypesenseDocument(minimal)
    expect(got.title).toBe("")
    expect(got.photo_url).toBe("")
    expect(got.city).toBe("")
    expect(got.country_code).toBe("")
    expect(got.pricing).toBeNull()
  })
})
