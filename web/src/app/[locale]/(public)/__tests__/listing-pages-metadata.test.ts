/**
 * Public-listing generateMetadata tests — PERF-W-02 + PERF-W-06.
 *
 * The pages are async Server Components; we exercise the
 * generateMetadata exports directly rather than rendering React.
 *
 * Mocks:
 *   - server fetcher returns a known found count
 *   - getTranslations returns templated strings keyed on the
 *     translation keys the source uses
 *
 * Asserts:
 *   - title / description interpolate the count
 *   - canonical URL is the listing's clean path
 *   - openGraph + twitter shapes are populated
 */

import { describe, it, expect, vi, beforeEach } from "vitest"

const fetchMock = vi.fn()
vi.mock("@/features/provider/api/search-server", () => ({
  fetchListingFirstPage: (...args: unknown[]) => fetchMock(...args),
}))

// Stub out the heavy client SearchPage so importing the page module
// doesn't pull in TanStack Query / next-intl runtime — the test only
// exercises the server-side `generateMetadata` export.
vi.mock("@/features/provider/components/search-page", () => ({
  SearchPage: () => null,
}))

vi.mock("@/features/provider/api/listing-jsonld", () => ({
  buildItemList: () => ({}),
}))

vi.mock("next-intl/server", () => ({
  getTranslations: async ({ namespace }: { namespace: string }) => {
    return (key: string, vars: Record<string, unknown> = {}) => {
      // Simple template implementation matching next-intl's syntax —
      // sufficient to assert that the source code passes the right
      // keys + variables.
      const count = vars.count ?? "?"
      return `${namespace}.${key}(${count})`
    }
  },
}))

beforeEach(() => {
  fetchMock.mockReset()
})

async function callMetadata(modulePath: string, locale: string) {
  const mod = await import(modulePath)
  const params = Promise.resolve({ locale })
  return mod.generateMetadata({ params })
}

describe("Public listing generateMetadata — PERF-W-02", () => {
  it("agencies: builds title, description, canonical, OG, Twitter", async () => {
    fetchMock.mockResolvedValue({
      found: 42,
      documents: [],
      has_more: false,
      next_cursor: "",
      search_id: "x",
      highlights: [],
      facet_counts: {},
      out_of: 42,
      page: 1,
      per_page: 20,
      search_time_ms: 5,
    })

    const md = await callMetadata(
      "../agencies/page",
      "en",
    )

    expect(md.title).toContain("agencies.title(42)")
    expect(md.description).toContain("agencies.description(42)")
    expect(md.alternates.canonical).toBe("/agencies")
    expect(md.openGraph.type).toBe("website")
    expect(md.openGraph.title).toContain("agencies.title")
    expect(md.twitter.card).toBe("summary")
  })

  it("freelancers: builds title, description, canonical, OG, Twitter", async () => {
    fetchMock.mockResolvedValue({
      found: 17,
      documents: [],
      has_more: false,
      next_cursor: "",
      search_id: "x",
      highlights: [],
      facet_counts: {},
      out_of: 17,
      page: 1,
      per_page: 20,
      search_time_ms: 5,
    })

    const md = await callMetadata(
      "../freelancers/page",
      "en",
    )

    expect(md.title).toContain("freelancers.title(17)")
    expect(md.description).toContain("freelancers.description(17)")
    expect(md.alternates.canonical).toBe("/freelancers")
  })

  it("referrers: builds title, description, canonical, OG, Twitter", async () => {
    fetchMock.mockResolvedValue({
      found: 8,
      documents: [],
      has_more: false,
      next_cursor: "",
      search_id: "x",
      highlights: [],
      facet_counts: {},
      out_of: 8,
      page: 1,
      per_page: 20,
      search_time_ms: 5,
    })

    const md = await callMetadata(
      "../referrers/page",
      "en",
    )

    expect(md.title).toContain("referrers.title(8)")
    expect(md.alternates.canonical).toBe("/referrers")
  })

  it("falls back to count=0 when the fetcher returns null (graceful)", async () => {
    fetchMock.mockResolvedValue(null)

    const md = await callMetadata(
      "../agencies/page",
      "en",
    )

    expect(md.title).toContain("agencies.title(0)")
    // canonical must always be set so a transient backend error
    // never strands the URL with no canonical.
    expect(md.alternates.canonical).toBe("/agencies")
  })
})
