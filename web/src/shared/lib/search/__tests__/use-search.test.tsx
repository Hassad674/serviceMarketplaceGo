/**
 * use-search.test.tsx exercises the TanStack Query hook end-to-end
 * with a mocked fetch implementation. We stub the network layer so
 * the test stays hermetic and runs in the standard vitest sandbox.
 */

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { renderHook, waitFor } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import type { ReactNode } from "react"
import { useSearch } from "../use-search"

const SCOPED_KEY_PAYLOAD = {
  key: "scoped-key-xyz",
  host: "http://localhost:8108",
  expires_at: Math.floor(Date.now() / 1000) + 3600,
  persona: "freelance" as const,
}

const TYPESENSE_PAYLOAD = {
  found: 1,
  out_of: 1,
  page: 1,
  per_page: 20,
  search_time_ms: 4,
  hits: [
    {
      document: {
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
      highlights: [
        { field: "display_name", snippet: "<mark>Alice</mark>" },
      ],
    },
  ],
  facet_counts: [
    {
      field_name: "skills",
      counts: [
        { value: "go", count: 12 },
        { value: "react", count: 8 },
      ],
    },
  ],
  request_params: {
    collection_name: "marketplace_actors",
    q: "alice",
    first_q: "alice",
    per_page: 20,
  },
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
    if (url.includes("/api/v1/search/key")) {
      return Promise.resolve({
        ok: true,
        status: 200,
        json: async () => SCOPED_KEY_PAYLOAD,
      })
    }
    if (url.includes("/collections/marketplace_actors/documents/search")) {
      return Promise.resolve({
        ok: true,
        status: 200,
        json: async () => TYPESENSE_PAYLOAD,
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
  it("returns documents + highlights + facet counts", async () => {
    const { result } = renderHook(
      () =>
        useSearch({
          persona: "freelance",
          query: "alice",
          filters: { skills: ["go"] },
          page: 1,
          perPage: 20,
        }),
      { wrapper },
    )
    await waitFor(() => expect(result.current.found).toBe(1))
    expect(result.current.documents[0]?.display_name).toBe("Alice")
    expect(result.current.highlights[0]?.["display_name"]).toBe("<mark>Alice</mark>")
    expect(result.current.facetCounts.skills?.go).toBe(12)
    expect(result.current.facetCounts.skills?.react).toBe(8)
  })

  it("does not fire fetch when persona is null", async () => {
    renderHook(
      () =>
        useSearch({
          persona: null,
          query: "",
          filters: {},
          page: 1,
        }),
      { wrapper },
    )
    // Give the hook a tick to settle then verify no /search/key call.
    await new Promise((r) => setTimeout(r, 30))
    const calls = fetchMock.mock.calls.map((c) => c[0] as string)
    expect(calls.some((u) => u.includes("/search/key"))).toBe(false)
  })

  it("forwards filter_by built from the SearchFilterInput", async () => {
    const { result } = renderHook(
      () =>
        useSearch({
          persona: "freelance",
          query: "*",
          filters: { skills: ["go"], languages: ["fr"] },
          page: 1,
        }),
      { wrapper },
    )
    await waitFor(() => expect(result.current.found).toBe(1))
    const searchCall = fetchMock.mock.calls.find((c) =>
      (c[0] as string).includes("/documents/search"),
    )
    const url = searchCall?.[0] as string
    expect(url).toContain("filter_by=")
    expect(decodeURIComponent(url)).toContain("languages_professional:[fr]")
    expect(decodeURIComponent(url)).toContain("skills:[go]")
  })
})
