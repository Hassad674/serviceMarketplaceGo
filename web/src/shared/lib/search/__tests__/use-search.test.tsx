/**
 * use-search.test.tsx exercises the TanStack Query hook end-to-end
 * with a mocked fetch. Phase 3: the hook talks to the backend proxy
 * `/api/v1/search`, NOT Typesense directly. The backend owns
 * embedding + analytics capture + cursor pagination, so the test
 * stubs the proxy response.
 */

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { renderHook, waitFor } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import type { ReactNode } from "react"
import { useSearch } from "../use-search"

const BACKEND_PAGE_1 = {
  search_id: "search-abc",
  documents: [
    {
      id: "11111111-1111-1111-1111-111111111111",
      persona: "freelance",
      is_published: true,
      display_name: "Alice",
      title: "Go Dev",
      photo_url: "",
      city: "Paris",
      country_code: "fr",
      location: [48.8566, 2.3522],
      work_mode: ["remote"],
      languages_professional: ["fr"],
      languages_conversational: [],
      availability_status: "available_now",
      availability_priority: 3,
      expertise_domains: ["dev"],
      skills: ["go"],
      skills_text: "go",
      pricing_negotiable: false,
      rating_average: 4.8,
      rating_count: 12,
      rating_score: 12.5,
      total_earned: 0,
      completed_projects: 5,
      profile_completion_score: 80,
      last_active_at: 1700000000,
      response_rate: 1,
      is_verified: true,
      is_top_rated: true,
      is_featured: false,
      created_at: 1700000000,
      updated_at: 1700000100,
    },
  ],
  highlights: [{ display_name: "<mark>Alice</mark>" }],
  facet_counts: { skills: { go: 12, react: 8 } },
  found: 1,
  out_of: 1,
  page: 1,
  per_page: 20,
  search_time_ms: 4,
  corrected_query: "",
  has_more: false,
}

function wrapper({ children }: { children: ReactNode }) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0, staleTime: 0 } },
  })
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>
}

let fetchMock: ReturnType<typeof vi.fn>

beforeEach(() => {
  fetchMock = vi.fn().mockImplementation((url: string) => {
    if (url.includes("/api/v1/search")) {
      return Promise.resolve({
        ok: true,
        status: 200,
        json: async () => BACKEND_PAGE_1,
      })
    }
    return Promise.reject(new Error(`unexpected fetch: ${url}`))
  })
  vi.stubGlobal("fetch", fetchMock)
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe("useSearch", () => {
  it("returns documents + highlights + facet counts + searchId", async () => {
    const { result } = renderHook(
      () =>
        useSearch({
          persona: "freelance",
          query: "alice",
          filters: { skills: ["go"] },
          perPage: 20,
        }),
      { wrapper },
    )
    await waitFor(() => expect(result.current.found).toBe(1))
    expect(result.current.documents[0]?.display_name).toBe("Alice")
    expect(result.current.highlights[0]?.["display_name"]).toBe("<mark>Alice</mark>")
    expect(result.current.facetCounts.skills?.go).toBe(12)
    expect(result.current.facetCounts.skills?.react).toBe(8)
    expect(result.current.searchId).toBe("search-abc")
    expect(result.current.hasMore).toBe(false)
  })

  it("does not fire fetch when persona is null", async () => {
    renderHook(
      () =>
        useSearch({
          persona: null,
          query: "",
          filters: {},
        }),
      { wrapper },
    )
    await new Promise((r) => setTimeout(r, 30))
    const calls = fetchMock.mock.calls.map((c) => c[0] as string)
    expect(calls.some((u) => u.includes("/api/v1/search"))).toBe(false)
  })

  it("unpacks filters into individual query params", async () => {
    const { result } = renderHook(
      () =>
        useSearch({
          persona: "freelance",
          query: "",
          filters: { skills: ["go"], languages: ["fr"] },
        }),
      { wrapper },
    )
    await waitFor(() => expect(result.current.found).toBe(1))
    const call = fetchMock.mock.calls.find((c) =>
      (c[0] as string).includes("/api/v1/search"),
    )
    const url = call?.[0] as string
    expect(url).toContain("persona=freelance")
    expect(decodeURIComponent(url)).toContain("skills=go")
    expect(decodeURIComponent(url)).toContain("languages=fr")
  })

  it("passes the cursor on subsequent pages", async () => {
    // Emit has_more=true with next_cursor, then an empty second page.
    let call = 0
    fetchMock.mockImplementation((url: string) => {
      call++
      if (!url.includes("/api/v1/search")) {
        return Promise.reject(new Error(`unexpected fetch: ${url}`))
      }
      if (call === 1) {
        return Promise.resolve({
          ok: true,
          status: 200,
          json: async () => ({
            ...BACKEND_PAGE_1,
            found: 40,
            has_more: true,
            next_cursor: "cursor-page-2",
          }),
        })
      }
      return Promise.resolve({
        ok: true,
        status: 200,
        json: async () => ({
          ...BACKEND_PAGE_1,
          documents: [],
          highlights: [],
          found: 40,
          page: 2,
          has_more: false,
          next_cursor: "",
        }),
      })
    })

    const { result } = renderHook(
      () =>
        useSearch({
          persona: "freelance",
          query: "go",
          filters: {},
        }),
      { wrapper },
    )
    await waitFor(() => expect(result.current.hasMore).toBe(true))
    result.current.loadMore()
    await waitFor(() => expect(result.current.hasMore).toBe(false))

    const secondCall = fetchMock.mock.calls[1]?.[0] as string
    expect(secondCall).toContain("cursor=cursor-page-2")
  })
})
