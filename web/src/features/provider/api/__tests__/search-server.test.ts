/**
 * search-server tests — PERF-W-02 server-side first-page fetcher.
 *
 * Covers:
 *   - calls the public /api/v1/search proxy with persona + per_page + sort_by
 *   - returns null on non-200 responses (graceful degradation)
 *   - returns null when fetch throws (network failure)
 *   - persona mapping per SearchType
 */

import { describe, it, expect, vi, afterEach } from "vitest"

vi.mock("@/shared/lib/api-client", () => ({
  API_BASE_URL: "http://localhost:8080",
}))

import { fetchListingFirstPage } from "../search-server"
import type { SearchServerPage } from "../search-server"

const sampleResponse: SearchServerPage = {
  search_id: "abc",
  documents: [],
  highlights: [],
  facet_counts: {},
  found: 0,
  out_of: 0,
  page: 1,
  per_page: 20,
  search_time_ms: 5,
  has_more: false,
}

afterEach(() => {
  vi.restoreAllMocks()
})

describe("fetchListingFirstPage — PERF-W-02 server fetcher", () => {
  it("hits /api/v1/search with persona=agency for agency listings", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => sampleResponse,
    })
    vi.stubGlobal("fetch", fetchMock)

    await fetchListingFirstPage("agency")
    const url = fetchMock.mock.calls[0][0] as string
    expect(url).toContain("persona=agency")
    expect(url).toContain("per_page=20")
    expect(url).toContain("sort_by=")
  })

  it("hits /api/v1/search with persona=freelance for freelancer listings", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => sampleResponse,
    })
    vi.stubGlobal("fetch", fetchMock)

    await fetchListingFirstPage("freelancer")
    const url = fetchMock.mock.calls[0][0] as string
    expect(url).toContain("persona=freelance")
  })

  it("hits /api/v1/search with persona=referrer for referrer listings", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => sampleResponse,
    })
    vi.stubGlobal("fetch", fetchMock)

    await fetchListingFirstPage("referrer")
    const url = fetchMock.mock.calls[0][0] as string
    expect(url).toContain("persona=referrer")
  })

  it("forwards the parsed response on 200", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ ...sampleResponse, found: 7 }),
    })
    vi.stubGlobal("fetch", fetchMock)

    const result = await fetchListingFirstPage("freelancer")
    expect(result?.found).toBe(7)
  })

  it("returns null on non-200 response (graceful)", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: false,
      json: async () => ({}),
    })
    vi.stubGlobal("fetch", fetchMock)

    const result = await fetchListingFirstPage("agency")
    expect(result).toBeNull()
  })

  it("returns null when fetch throws", async () => {
    const fetchMock = vi.fn().mockRejectedValue(new Error("network down"))
    vi.stubGlobal("fetch", fetchMock)

    const result = await fetchListingFirstPage("freelancer")
    expect(result).toBeNull()
  })

  it("uses ISR via next.revalidate=60 on the underlying fetch call", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => sampleResponse,
    })
    vi.stubGlobal("fetch", fetchMock)

    await fetchListingFirstPage("freelancer")
    const init = fetchMock.mock.calls[0][1] as { next?: { revalidate?: number } }
    expect(init.next?.revalidate).toBe(60)
  })
})
