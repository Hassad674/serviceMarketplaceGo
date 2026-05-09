import { describe, it, expect, vi, beforeEach } from "vitest"
import { render, screen } from "@testing-library/react"
import { NextIntlClientProvider } from "next-intl"
import messages from "@/../messages/en.json"
import { SearchPage } from "../search-page"
import type { RawSearchDocument } from "@/shared/lib/search/typesense-client"

/**
 * search-page-initial-query.test.tsx covers the new `initialQuery`
 * prop that lets the public listing routes (/freelancers, /agencies,
 * /referrers) seed the search input with the `?q=` URL param the
 * landing page passes through. Two contracts to lock:
 *  1. initialQuery non-empty -> the hook is called with the query and
 *     the input renders the prefilled value
 *  2. initialQuery non-empty -> the SSR `initialFirstPage` seed is
 *     dropped (it was prefetched without the query)
 */

vi.mock("@i18n/navigation", () => ({
  Link: ({
    href,
    children,
    ...rest
  }: {
    href: string
    children: React.ReactNode
    className?: string
  }) => (
    <a href={href} {...rest}>
      {children}
    </a>
  ),
}))

vi.mock("next/image", () => ({
  default: ({
    src,
    alt,
    ...rest
  }: {
    src: string
    alt: string
    width: number
    height: number
    className?: string
  // eslint-disable-next-line @next/next/no-img-element -- test mock substituting next/image
  }) => <img src={src} alt={alt} {...rest} />,
}))

const mockUseSearch = vi.fn()
vi.mock("@/shared/lib/search/use-search", () => ({
  useSearch: (args: unknown) => mockUseSearch(args),
}))

vi.mock("@/shared/lib/search/track-click", () => ({
  trackSearchClick: vi.fn(),
}))

beforeEach(() => {
  mockUseSearch.mockReset()
})

function makeResult(documents: RawSearchDocument[] = []) {
  return {
    documents,
    found: documents.length,
    page: 1,
    perPage: 20,
    facetCounts: {},
    highlights: [],
    searchId: "deterministic-search-id",
    correctedQuery: "",
    hasMore: false,
    isLoading: false,
    isFetchingMore: false,
    error: null,
    loadMore: vi.fn(),
    refetch: vi.fn(),
  }
}

function renderPage(props: {
  type?: "freelancer" | "agency" | "referrer"
  initialQuery?: string
  initialFirstPage?: { documents: RawSearchDocument[]; found: number } | null
} = {}) {
  return render(
    <NextIntlClientProvider locale="en" messages={messages}>
      <SearchPage
        type={props.type ?? "freelancer"}
        initialQuery={props.initialQuery}
        // The runtime accepts the wider BackendSearchPage but the
        // initial-query branch only inspects truthiness — passing
        // an unknown shape is acceptable for this contract test.
        initialFirstPage={
          // eslint-disable-next-line @typescript-eslint/no-explicit-any -- partial seed for test
          props.initialFirstPage as any
        }
      />
    </NextIntlClientProvider>,
  )
}

describe("SearchPage initialQuery prop", () => {
  it("seeds the search input with the initialQuery value", () => {
    mockUseSearch.mockReturnValue(makeResult())
    renderPage({ initialQuery: "designer" })
    const inputs = screen.getAllByDisplayValue("designer")
    expect(inputs.length).toBeGreaterThan(0)
  })

  it("calls useSearch with the initialQuery on first render", () => {
    mockUseSearch.mockReturnValue(makeResult())
    renderPage({ initialQuery: "stripe developer" })
    expect(mockUseSearch).toHaveBeenCalled()
    const arg = mockUseSearch.mock.calls[0][0] as { query: string }
    expect(arg.query).toBe("stripe developer")
  })

  it("falls back to empty string when initialQuery is omitted", () => {
    mockUseSearch.mockReturnValue(makeResult())
    renderPage({})
    const arg = mockUseSearch.mock.calls[0][0] as { query: string }
    expect(arg.query).toBe("")
  })

  it("drops the initialFirstPage seed when the initialQuery is non-empty", () => {
    mockUseSearch.mockReturnValue(makeResult())
    const seed = { documents: [], found: 0 }
    renderPage({ initialQuery: "designer", initialFirstPage: seed })
    const arg = mockUseSearch.mock.calls[0][0] as {
      initialFirstPage?: unknown
    }
    expect(arg.initialFirstPage).toBeUndefined()
  })

  it("keeps the initialFirstPage seed when the initialQuery is empty", () => {
    mockUseSearch.mockReturnValue(makeResult())
    const seed = { documents: [], found: 0 }
    renderPage({ initialQuery: "", initialFirstPage: seed })
    const arg = mockUseSearch.mock.calls[0][0] as {
      initialFirstPage?: unknown
    }
    expect(arg.initialFirstPage).toBe(seed)
  })
})
