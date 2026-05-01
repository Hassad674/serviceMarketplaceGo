import { describe, it, expect, vi, beforeEach } from "vitest"
import { render, screen } from "@testing-library/react"
import { NextIntlClientProvider } from "next-intl"
import messages from "@/../messages/en.json"
import { SearchPage } from "../search-page"
import type { RawSearchDocument } from "@/shared/lib/search/typesense-client"

/**
 * search-page.test.tsx covers the provider-feature SearchPage,
 * which is now a pure Typesense adapter (phase 4 retired the SQL
 * fallback). The test mocks the shared useSearch hook and asserts
 * that the page renders the documents it receives, without pinning
 * the specific Typesense request shape — that's covered by
 * use-search.test.tsx upstream.
 */

// Mock next-intl navigation (Link used inside ProviderCard)
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

// Mock the useSearch hook at module boundary so the component
// receives a deterministic result set per test.
const mockUseSearch = vi.fn()
vi.mock("@/shared/lib/search/use-search", () => ({
  useSearch: (args: unknown) => mockUseSearch(args),
}))

// Track-click is fire-and-forget via navigator.sendBeacon — mock so
// the tests do not hit network.
vi.mock("@/shared/lib/search/track-click", () => ({
  trackSearchClick: vi.fn(),
}))

function createDoc(overrides: Partial<RawSearchDocument> = {}): RawSearchDocument {
  // RawSearchDocument mirrors the Typesense wire format. Only the
  // fields the card actually reads are pinned; the rest come via
  // overrides for the specific test case.
  return {
    id: "org-1:freelance",
    organization_id: "org-1",
    persona: "freelance",
    is_published: true,
    display_name: "Test Freelance",
    title: "Developer",
    work_mode: [],
    languages_professional: [],
    languages_conversational: [],
    availability_status: "available_now",
    availability_priority: 3,
    expertise_domains: [],
    skills: [],
    skills_text: "",
    pricing_type: "",
    pricing_min_amount: 0,
    pricing_max_amount: 0,
    pricing_currency: "EUR",
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
  } as RawSearchDocument
}

function mockSearchResult(
  documents: RawSearchDocument[],
  overrides: Record<string, unknown> = {},
) {
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
    ...overrides,
  }
}

function renderPage(type: "freelancer" | "agency" | "referrer") {
  return render(
    <NextIntlClientProvider locale="en" messages={messages}>
      <SearchPage type={type} />
    </NextIntlClientProvider>,
  )
}

describe("SearchPage (Typesense-only)", () => {
  beforeEach(() => {
    mockUseSearch.mockReset()
  })

  it("renders the documents returned by useSearch", () => {
    mockUseSearch.mockReturnValue(
      mockSearchResult([
        createDoc({ display_name: "Alice", title: "React dev" }),
        createDoc({ organization_id: "org-2", display_name: "Bob" }),
      ]),
    )
    renderPage("freelancer")
    expect(screen.getByText(/Alice/)).toBeInTheDocument()
    expect(screen.getByText(/Bob/)).toBeInTheDocument()
  })

  it("shows loading state when the hook is loading", () => {
    mockUseSearch.mockReturnValue(mockSearchResult([], { isLoading: true }))
    const { container } = renderPage("freelancer")
    // SearchPageLayout renders skeleton cards (animate-shimmer) while loading.
    expect(container.querySelectorAll(".animate-shimmer").length).toBeGreaterThan(0)
  })

  it("maps the feature-level type to the right persona", () => {
    mockUseSearch.mockReturnValue(mockSearchResult([]))
    renderPage("agency")
    expect(mockUseSearch).toHaveBeenCalledTimes(1)
    const arg = mockUseSearch.mock.calls[0][0] as { persona: string }
    expect(arg.persona).toBe("agency")
  })

  it("renders the did-you-mean banner when useSearch returns a corrected query", () => {
    mockUseSearch.mockReturnValue(
      mockSearchResult([], { correctedQuery: "react-dev" }),
    )
    const { container } = renderPage("freelancer")
    // The banner is the only element with role=status rendered by
    // DidYouMeanBanner; assert it contains the corrected text so the
    // check is decoupled from the banner's surrounding copy.
    const banner = container.querySelector("[role=status]")
    expect(banner).not.toBeNull()
    expect(banner?.textContent).toMatch(/react-dev/)
  })
})
