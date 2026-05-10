import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import {
  fetchEnterpriseApplicationsStats,
  fetchKeywordStats,
  fetchVisibilityStats,
} from "../api/stats-api"

// stats-api unit tests — exercise the URL builders, query-string
// clamping, and the response normalisation logic. The fetch layer is
// shimmed via globalThis.fetch so the tests run pure (no msw / jsdom
// network) and so the API contract changes (envelope shape, defaults)
// fail the suite loudly.

const captured: { url: string; init: RequestInit | undefined }[] = []

beforeEach(() => {
  captured.length = 0
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      captured.push({ url: input.toString(), init })
      return new Response(JSON.stringify({ data: { series: [], total_views: 0 } }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      })
    }),
  )
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe("fetchVisibilityStats", () => {
  it("requests /api/v1/me/stats/visibility with the days param", async () => {
    await fetchVisibilityStats(7)
    expect(captured[0].url).toMatch(/\/api\/v1\/me\/stats\/visibility\?days=7$/)
  })

  it("normalises an empty payload to safe defaults", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => new Response(JSON.stringify({ data: {} }), { status: 200 })),
    )
    const got = await fetchVisibilityStats(30)
    expect(got).toEqual({
      organization_id: "",
      period_days: 0,
      total_views: 0,
      unique_viewers: 0,
      search_appearances: 0,
      avg_search_position: null,
      series: [],
    })
  })

  it("preserves a non-numeric avg_search_position as null", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(
        async () =>
          new Response(
            JSON.stringify({ data: { avg_search_position: null, series: [] } }),
            { status: 200 },
          ),
      ),
    )
    const got = await fetchVisibilityStats(30)
    expect(got.avg_search_position).toBeNull()
  })
})

describe("fetchKeywordStats", () => {
  it("clamps the limit to [1..100]", async () => {
    await fetchKeywordStats(30, 9999)
    expect(captured[0].url).toMatch(/limit=100/)
  })

  it("rejects fractional limits", async () => {
    await fetchKeywordStats(30, 7.7)
    expect(captured[0].url).toMatch(/limit=7/)
  })

  it("returns an empty array on non-array data", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => new Response(JSON.stringify({ data: null }), { status: 200 })),
    )
    const got = await fetchKeywordStats(30)
    expect(got).toEqual([])
  })

  it("treats limits below 1 as 1", async () => {
    await fetchKeywordStats(7, 0)
    expect(captured[0].url).toMatch(/limit=1/)
  })
})

describe("fetchEnterpriseApplicationsStats", () => {
  it("requests the right path", async () => {
    await fetchEnterpriseApplicationsStats(90)
    expect(captured[0].url).toMatch(
      /\/api\/v1\/me\/stats\/enterprise-applications\?days=90$/,
    )
  })

  it("normalises missing fields", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => new Response(JSON.stringify({ data: {} }), { status: 200 })),
    )
    const got = await fetchEnterpriseApplicationsStats(7)
    expect(got).toEqual({
      organization_id: "",
      period_days: 0,
      total_count: 0,
      series: [],
    })
  })
})
