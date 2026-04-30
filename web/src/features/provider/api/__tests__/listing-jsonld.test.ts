/**
 * listing-jsonld tests — PERF-W-02 ItemList structured data.
 *
 * Covers:
 *   - schema shape (Schema.org ItemList)
 *   - per-persona @type mapping (Person vs Organization)
 *   - URL/path mapping per type
 *   - graceful handling of missing fields
 */

import { describe, it, expect } from "vitest"
import { buildItemList } from "../listing-jsonld"
import type { RawSearchDocument } from "@/shared/lib/search/typesense-client"

function makeDoc(overrides: Partial<RawSearchDocument> = {}): RawSearchDocument {
  return {
    id: "org-1:freelance",
    organization_id: "org-1",
    persona: "freelance",
    is_published: true,
    display_name: "Alice",
    title: "Go developer",
    photo_url: "https://r2/avatar.jpg",
    city: "Paris",
    country_code: "FR",
    work_mode: ["remote"],
    languages_professional: ["fr"],
    languages_conversational: [],
    availability_status: "available_now",
    availability_priority: 1,
    expertise_domains: ["software"],
    skills: ["go"],
    skills_text: "go",
    pricing_type: "daily",
    pricing_min_amount: 50000,
    pricing_max_amount: null as unknown as number,
    pricing_currency: "EUR",
    pricing_negotiable: false,
    rating_average: 4.8,
    rating_count: 12,
    rating_score: 4.8,
    total_earned: 100000,
    completed_projects: 5,
    profile_completion_score: 0.9,
    last_active_at: 1700000000,
    response_rate: 0.9,
    is_verified: true,
    is_top_rated: false,
    is_featured: false,
    created_at: 1700000000,
    updated_at: 1700000000,
    ...overrides,
  }
}

describe("buildItemList — PERF-W-02 ItemList structured data", () => {
  it("produces a Schema.org ItemList envelope", () => {
    const result = buildItemList({
      type: "freelancer",
      documents: [makeDoc()],
      totalFound: 42,
    })

    expect(result["@context"]).toBe("https://schema.org")
    expect(result["@type"]).toBe("ItemList")
    expect(result.numberOfItems).toBe(42)
    expect(result.name).toBe("Freelancers")
  })

  it("maps freelancer documents to Person items pointing at /freelancers/:id", () => {
    const result = buildItemList({
      type: "freelancer",
      documents: [makeDoc({ organization_id: "org-77" })],
      totalFound: 1,
    })

    const items = result.itemListElement as Array<Record<string, unknown>>
    expect(items).toHaveLength(1)
    expect(items[0].position).toBe(1)
    const item = items[0].item as Record<string, unknown>
    expect(item["@type"]).toBe("Person")
    expect(item["@id"]).toBe("/freelancers/org-77")
    expect(item.name).toBe("Alice")
  })

  it("maps agency documents to Organization items pointing at /agencies/:id", () => {
    const result = buildItemList({
      type: "agency",
      documents: [makeDoc({ organization_id: "agency-9", persona: "agency" })],
      totalFound: 1,
    })

    const items = result.itemListElement as Array<Record<string, unknown>>
    const item = items[0].item as Record<string, unknown>
    expect(item["@type"]).toBe("Organization")
    expect(item["@id"]).toBe("/agencies/agency-9")
  })

  it("maps referrer documents to Person items pointing at /referrers/:id", () => {
    const result = buildItemList({
      type: "referrer",
      documents: [makeDoc({ organization_id: "ref-3", persona: "referrer" })],
      totalFound: 1,
    })

    const items = result.itemListElement as Array<Record<string, unknown>>
    const item = items[0].item as Record<string, unknown>
    expect(item["@type"]).toBe("Person")
    expect(item["@id"]).toBe("/referrers/ref-3")
  })

  it("preserves position ordering across multiple docs", () => {
    const result = buildItemList({
      type: "freelancer",
      documents: [
        makeDoc({ id: "a:freelance", organization_id: "a", display_name: "A" }),
        makeDoc({ id: "b:freelance", organization_id: "b", display_name: "B" }),
        makeDoc({ id: "c:freelance", organization_id: "c", display_name: "C" }),
      ],
      totalFound: 3,
    })

    const items = result.itemListElement as Array<Record<string, unknown>>
    expect(items.map((it) => it.position)).toEqual([1, 2, 3])
    expect(
      items.map((it) => (it.item as Record<string, unknown>).name),
    ).toEqual(["A", "B", "C"])
  })

  it("falls back to id-prefix parsing when organization_id is missing", () => {
    const result = buildItemList({
      type: "freelancer",
      documents: [
        makeDoc({
          id: "raw-uuid:freelance",
          organization_id: undefined,
        }),
      ],
      totalFound: 1,
    })

    const items = result.itemListElement as Array<Record<string, unknown>>
    const item = items[0].item as Record<string, unknown>
    expect(item["@id"]).toBe("/freelancers/raw-uuid")
  })

  it("omits optional fields when source data is missing", () => {
    const result = buildItemList({
      type: "freelancer",
      documents: [
        makeDoc({
          city: "",
          country_code: "",
          photo_url: "",
          title: "",
        }),
      ],
      totalFound: 1,
    })

    const items = result.itemListElement as Array<Record<string, unknown>>
    const item = items[0].item as Record<string, unknown>
    expect(item.address).toBeUndefined()
    expect(item.image).toBeUndefined()
    expect(item.description).toBeUndefined()
  })

  it("includes a PostalAddress when city is present", () => {
    const result = buildItemList({
      type: "freelancer",
      documents: [
        makeDoc({ city: "Lyon", country_code: "FR" }),
      ],
      totalFound: 1,
    })

    const items = result.itemListElement as Array<Record<string, unknown>>
    const item = items[0].item as Record<string, unknown>
    expect(item.address).toEqual({
      "@type": "PostalAddress",
      addressLocality: "Lyon",
      addressCountry: "FR",
    })
  })
})
