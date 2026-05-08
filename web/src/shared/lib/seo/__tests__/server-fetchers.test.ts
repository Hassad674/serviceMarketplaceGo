import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"

import {
  fetchPublicReviews,
  fetchPublicAverageRating,
  pickRelatedProfiles,
} from "../server-fetchers"
import type { RawSearchDocument } from "@/shared/lib/search/typesense-client"

const originalFetch = global.fetch

beforeEach(() => {
  global.fetch = vi.fn() as unknown as typeof fetch
})

afterEach(() => {
  global.fetch = originalFetch
  vi.restoreAllMocks()
})

function mockFetch(response: { ok: boolean; data?: unknown }) {
  ;(global.fetch as unknown as ReturnType<typeof vi.fn>).mockResolvedValueOnce({
    ok: response.ok,
    json: async () => response.data,
  })
}

describe("fetchPublicReviews", () => {
  it("returns the data array when the endpoint responds 200", async () => {
    mockFetch({
      ok: true,
      data: { data: [{ id: "rev-1" }, { id: "rev-2" }] },
    })
    const out = await fetchPublicReviews("org-1", 5)
    expect(out).toHaveLength(2)
  })

  it("returns null when the endpoint responds non-2xx", async () => {
    mockFetch({ ok: false })
    expect(await fetchPublicReviews("org-1")).toBeNull()
  })

  it("returns null when fetch throws (transient backend hiccup)", async () => {
    ;(global.fetch as unknown as ReturnType<typeof vi.fn>).mockRejectedValueOnce(
      new Error("ECONNREFUSED"),
    )
    expect(await fetchPublicReviews("org-1")).toBeNull()
  })

  it("returns null when the response shape is unexpected", async () => {
    mockFetch({ ok: true, data: { wrong: "shape" } })
    expect(await fetchPublicReviews("org-1")).toBeNull()
  })
})

describe("fetchPublicAverageRating", () => {
  it("returns the AverageRating envelope on success", async () => {
    mockFetch({ ok: true, data: { data: { average: 4.5, count: 12 } } })
    const out = await fetchPublicAverageRating("org-1")
    expect(out).toEqual({ average: 4.5, count: 12 })
  })

  it("returns null on non-2xx", async () => {
    mockFetch({ ok: false })
    expect(await fetchPublicAverageRating("org-1")).toBeNull()
  })
})

describe("pickRelatedProfiles", () => {
  function makeDoc(overrides: Partial<RawSearchDocument>): RawSearchDocument {
    return {
      id: "doc-1",
      organization_id: "doc-1",
      persona: "freelance",
      is_published: true,
      display_name: "Doc",
      work_mode: [],
      languages_professional: [],
      languages_conversational: [],
      availability_status: "available_now",
      availability_priority: 0,
      expertise_domains: [],
      skills: [],
      skills_text: "",
      pricing_negotiable: false,
      rating_average: 0,
      rating_count: 0,
      rating_score: 0,
      total_earned: 0,
      completed_projects: 0,
      profile_completion_score: 0,
      last_active_at: 0,
      response_rate: 0,
      is_verified: false,
      is_top_rated: false,
      is_featured: false,
      created_at: 0,
      updated_at: 0,
      ...overrides,
    }
  }

  it("excludes the current org from candidates", () => {
    const out = pickRelatedProfiles({
      candidates: [
        makeDoc({ id: "self", organization_id: "self" }),
        makeDoc({ id: "other", organization_id: "other" }),
      ],
      excludeOrgId: "self",
      primaryExpertise: undefined,
      city: undefined,
      limit: 6,
    })
    expect(out).toHaveLength(1)
    expect(out[0].organization_id).toBe("other")
  })

  it("ranks expertise match higher than city match", () => {
    const out = pickRelatedProfiles({
      candidates: [
        makeDoc({
          id: "city-only",
          organization_id: "city-only",
          city: "Paris",
        }),
        makeDoc({
          id: "expertise-only",
          organization_id: "expertise-only",
          expertise_domains: ["web"],
        }),
      ],
      excludeOrgId: "self",
      primaryExpertise: "web",
      city: "Paris",
      limit: 6,
    })
    expect(out[0].organization_id).toBe("expertise-only")
  })

  it("respects the limit", () => {
    const out = pickRelatedProfiles({
      candidates: Array.from({ length: 10 }, (_, i) =>
        makeDoc({ id: `d-${i}`, organization_id: `d-${i}` }),
      ),
      excludeOrgId: "self",
      primaryExpertise: undefined,
      city: undefined,
      limit: 3,
    })
    expect(out).toHaveLength(3)
  })
})
