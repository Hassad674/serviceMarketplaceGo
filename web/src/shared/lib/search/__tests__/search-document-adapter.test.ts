import { describe, expect, it } from "vitest"
import {
  inferPersona,
  toSearchDocument,
} from "../search-document-adapter"

describe("inferPersona", () => {
  it("prefers the explicit hint when provided", () => {
    expect(inferPersona("provider_personal", "referrer")).toBe("referrer")
    expect(inferPersona("agency", "freelance")).toBe("freelance")
  })

  it("maps agency org type to agency persona", () => {
    expect(inferPersona("agency", undefined)).toBe("agency")
  })

  it("defaults provider_personal to freelance persona", () => {
    expect(inferPersona("provider_personal", undefined)).toBe("freelance")
  })

  it("defaults unknown org types to freelance persona", () => {
    expect(inferPersona("enterprise", undefined)).toBe("freelance")
    expect(inferPersona(undefined, undefined)).toBe("freelance")
  })
})

describe("toSearchDocument", () => {
  it("fills every required field from a minimal payload", () => {
    const doc = toSearchDocument(
      {
        organization_id: "org-1",
        name: "Alice",
      },
      "freelance",
    )

    expect(doc.id).toBe("org-1")
    expect(doc.persona).toBe("freelance")
    expect(doc.display_name).toBe("Alice")
    expect(doc.title).toBe("")
    expect(doc.photo_url).toBe("")
    expect(doc.city).toBe("")
    expect(doc.country_code).toBe("")
    expect(doc.languages_professional).toEqual([])
    expect(doc.availability_status).toBe("available_now")
    expect(doc.expertise_domains).toEqual([])
    expect(doc.skills).toEqual([])
    expect(doc.pricing).toBeNull()
    expect(doc.rating).toEqual({ average: 0, count: 0 })
    expect(doc.total_earned).toBe(0)
    expect(doc.completed_projects).toBe(0)
  })

  it("maps the full legacy summary shape into the document", () => {
    const doc = toSearchDocument(
      {
        organization_id: "org-1",
        name: "Acme Studio",
        org_type: "agency",
        title: "Boutique design",
        photo_url: "https://cdn.example/a.jpg",
        city: "Paris",
        country_code: "FR",
        languages_professional: ["fr", "en"],
        availability_status: "available_soon",
        skills: [
          { display_text: "React", skill_text: "react" },
          { display_text: "Go", skill_text: "go" },
        ],
        pricing: [
          {
            kind: "direct",
            type: "daily",
            min_amount: 60000,
            max_amount: null,
            currency: "EUR",
            negotiable: true,
          },
        ],
        average_rating: 4.8,
        review_count: 12,
        total_earned: 1500000,
        completed_projects: 24,
        created_at: "2026-02-01T00:00:00Z",
      },
      "agency",
    )

    expect(doc.persona).toBe("agency")
    expect(doc.display_name).toBe("Acme Studio")
    expect(doc.city).toBe("Paris")
    expect(doc.country_code).toBe("FR")
    expect(doc.languages_professional).toEqual(["fr", "en"])
    expect(doc.availability_status).toBe("available_soon")
    expect(doc.skills).toEqual(["React", "Go"])
    expect(doc.pricing).toEqual({
      type: "daily",
      min_amount: 60000,
      max_amount: null,
      currency: "EUR",
      negotiable: true,
    })
    expect(doc.rating).toEqual({ average: 4.8, count: 12 })
    expect(doc.total_earned).toBe(1500000)
    expect(doc.completed_projects).toBe(24)
  })

  it("picks the referral pricing kind for referrer persona", () => {
    const doc = toSearchDocument(
      {
        organization_id: "org-1",
        name: "Referrer Rita",
        pricing: [
          {
            kind: "direct",
            type: "daily",
            min_amount: 40000,
            max_amount: null,
            currency: "EUR",
          },
          {
            kind: "referral",
            type: "commission_pct",
            min_amount: 500,
            max_amount: 1500,
            currency: "pct",
          },
        ],
      },
      "referrer",
    )

    expect(doc.pricing?.type).toBe("commission_pct")
    expect(doc.pricing?.currency).toBe("pct")
  })

  it("coerces unknown availability values to available_now", () => {
    const doc = toSearchDocument(
      { organization_id: "o", availability_status: "invalid" },
      "freelance",
    )
    expect(doc.availability_status).toBe("available_now")
  })

  it("caps skills at 6 entries and skips empty labels", () => {
    const doc = toSearchDocument(
      {
        organization_id: "o",
        skills: [
          { display_text: "A" },
          { display_text: "" },
          { display_text: "B" },
          { display_text: "C" },
          { display_text: "D" },
          { display_text: "E" },
          { display_text: "F" },
          { display_text: "G" },
        ],
      },
      "freelance",
    )
    expect(doc.skills).toHaveLength(6)
    expect(doc.skills).toEqual(["A", "B", "C", "D", "E", "F"])
  })
})
