/**
 * freelancers-public-search.test.tsx asserts the public listing
 * routes forward the `?q=` URL parameter from the landing search bar
 * into the client SearchPage component as `initialQuery`.
 *
 * This is the regression guard for the bug where typing a query in
 * the landing hero ("react"), submitting, and landing on
 * /freelancers?q=react resulted in skeletons that never resolved
 * because the initial query was lost between the page boundary and
 * the search hook.
 *
 * The test exercises every public listing route (`/freelancers`,
 * `/agencies`, `/referrers`) since they all share the same
 * `readInitialQuery` helper shape.
 */

import { describe, it, expect, vi, beforeEach } from "vitest"
import { render } from "@testing-library/react"

const fetchMock = vi.fn()
vi.mock("@/features/provider/api/search-server", () => ({
  fetchListingFirstPage: (...args: unknown[]) => fetchMock(...args),
}))

const searchPageProps: Array<Record<string, unknown>> = []
vi.mock("@/features/provider/components/search-page", () => ({
  SearchPage: (props: Record<string, unknown>) => {
    searchPageProps.push(props)
    return null
  },
}))

vi.mock("@/features/provider/api/listing-jsonld", () => ({
  buildItemList: () => null,
}))

vi.mock("@/shared/lib/json-ld", () => ({
  safeJsonLd: () => "{}",
}))

vi.mock("next-intl/server", () => ({
  getTranslations: async () => (key: string) => key,
}))

const emptyFirstPage = {
  found: 0,
  documents: [],
  has_more: false,
  next_cursor: "",
  search_id: "",
  highlights: [],
  facet_counts: {},
  out_of: 0,
  page: 1,
  per_page: 20,
  search_time_ms: 0,
}

beforeEach(() => {
  fetchMock.mockReset()
  searchPageProps.length = 0
  fetchMock.mockResolvedValue(emptyFirstPage)
})

async function renderPage(modulePath: string, q: string | undefined) {
  const mod = await import(modulePath)
  const params = Promise.resolve({ locale: "fr" })
  const searchParams = Promise.resolve(q !== undefined ? { q } : {})
  // The default export is an async Server Component. Awaiting it
  // yields a ready-to-render JSX tree; passing that tree to
  // @testing-library/react actually mounts the children (including
  // the mocked SearchPage), which is when its props are captured.
  const element = await mod.default({ params, searchParams })
  render(element)
}

describe("Public listing pages — q param propagation", () => {
  it("/freelancers forwards ?q=react to SearchPage as initialQuery", async () => {
    await renderPage("../freelancers/page", "react")

    expect(searchPageProps).toHaveLength(1)
    expect(searchPageProps[0]?.type).toBe("freelancer")
    expect(searchPageProps[0]?.initialQuery).toBe("react")
  })

  it("/agencies forwards ?q=design to SearchPage as initialQuery", async () => {
    await renderPage("../agencies/page", "design")

    expect(searchPageProps).toHaveLength(1)
    expect(searchPageProps[0]?.type).toBe("agency")
    expect(searchPageProps[0]?.initialQuery).toBe("design")
  })

  it("/referrers forwards ?q=consultant to SearchPage as initialQuery", async () => {
    await renderPage("../referrers/page", "consultant")

    expect(searchPageProps).toHaveLength(1)
    expect(searchPageProps[0]?.type).toBe("referrer")
    expect(searchPageProps[0]?.initialQuery).toBe("consultant")
  })

  it("missing ?q= falls back to empty string (unscoped catalog)", async () => {
    await renderPage("../freelancers/page", undefined)

    expect(searchPageProps).toHaveLength(1)
    expect(searchPageProps[0]?.initialQuery).toBe("")
  })

  it("empty ?q= keeps initialQuery empty so the unscoped seed is used", async () => {
    await renderPage("../freelancers/page", "")

    expect(searchPageProps).toHaveLength(1)
    expect(searchPageProps[0]?.initialQuery).toBe("")
  })
})
