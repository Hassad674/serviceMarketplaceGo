import { describe, it, expect, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import { NextIntlClientProvider } from "next-intl"
import messages from "@/../messages/en.json"
import { SearchPage } from "../search-page"
import type { PublicProfileSummary } from "../../api/search-api"

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

// Mock next/image (used inside ProviderCard)
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
  }) => <img src={src} alt={alt} {...rest} />,
}))

// Mock the useSearchProfiles hook
const mockUseSearchProfiles = vi.fn()
vi.mock("../../hooks/use-search", () => ({
  useSearchProfiles: (...args: unknown[]) => mockUseSearchProfiles(...args),
}))

function createProfile(
  overrides: Partial<PublicProfileSummary> = {},
): PublicProfileSummary {
  return {
    user_id: "user-1",
    display_name: "Test User",
    first_name: "Test",
    last_name: "User",
    role: "provider",
    title: "Developer",
    photo_url: "",
    referrer_enabled: false,
    average_rating: 0,
    review_count: 0,
    ...overrides,
  }
}

function mockInfiniteResult(
  profiles: PublicProfileSummary[],
  overrides: Record<string, unknown> = {},
) {
  return {
    data: {
      pages: [{ data: profiles, next_cursor: "", has_more: false }],
      pageParams: [undefined],
    },
    isLoading: false,
    error: null,
    fetchNextPage: vi.fn(),
    hasNextPage: false,
    isFetchingNextPage: false,
    ...overrides,
  }
}

function renderSearchPage(type: "freelancer" | "agency" | "referrer") {
  return render(
    <NextIntlClientProvider locale="en" messages={messages}>
      <SearchPage type={type} />
    </NextIntlClientProvider>,
  )
}

describe("SearchPage", () => {
  it("shows loading skeleton when data is loading", () => {
    mockUseSearchProfiles.mockReturnValue(
      mockInfiniteResult([], {
        data: undefined,
        isLoading: true,
      }),
    )

    renderSearchPage("freelancer")

    // Skeleton renders 6 placeholder items with animate-pulse divs
    const pulseElements = document.querySelectorAll(".animate-pulse")
    expect(pulseElements.length).toBeGreaterThan(0)
  })

  it("shows empty state when no results", () => {
    mockUseSearchProfiles.mockReturnValue(mockInfiniteResult([]))

    renderSearchPage("freelancer")

    expect(screen.getByText(messages.search.noResults)).toBeInTheDocument()
    expect(
      screen.getByText(messages.search.noResultsDesc),
    ).toBeInTheDocument()
  })

  it("shows provider cards when results exist", () => {
    const profiles = [
      createProfile({
        user_id: "1",
        first_name: "Alice",
        last_name: "Smith",
        title: "UX Designer",
      }),
      createProfile({
        user_id: "2",
        first_name: "Bob",
        last_name: "Jones",
        title: "Backend Dev",
      }),
    ]

    mockUseSearchProfiles.mockReturnValue(mockInfiniteResult(profiles))

    renderSearchPage("freelancer")

    expect(screen.getByText("Alice Smith")).toBeInTheDocument()
    expect(screen.getByText("Bob Jones")).toBeInTheDocument()
    expect(screen.getByText("UX Designer")).toBeInTheDocument()
    expect(screen.getByText("Backend Dev")).toBeInTheDocument()
  })

  it("displays correct title for freelancer type", () => {
    mockUseSearchProfiles.mockReturnValue(mockInfiniteResult([]))

    renderSearchPage("freelancer")

    expect(
      screen.getByRole("heading", { name: messages.search.findFreelancers }),
    ).toBeInTheDocument()
  })

  it("displays correct title for agency type", () => {
    mockUseSearchProfiles.mockReturnValue(mockInfiniteResult([]))

    renderSearchPage("agency")

    expect(
      screen.getByRole("heading", { name: messages.search.findAgencies }),
    ).toBeInTheDocument()
  })

  it("displays correct title for referrer type", () => {
    mockUseSearchProfiles.mockReturnValue(mockInfiniteResult([]))

    renderSearchPage("referrer")

    expect(
      screen.getByRole("heading", { name: messages.search.findReferrers }),
    ).toBeInTheDocument()
  })

  it("shows error message when loading fails", () => {
    mockUseSearchProfiles.mockReturnValue(
      mockInfiniteResult([], {
        data: undefined,
        error: new Error("Network error"),
      }),
    )

    renderSearchPage("freelancer")

    expect(
      screen.getByText(messages.search.errorLoading),
    ).toBeInTheDocument()
  })

  it("passes correct type to useSearchProfiles hook", () => {
    mockUseSearchProfiles.mockReturnValue(mockInfiniteResult([]))

    renderSearchPage("agency")

    expect(mockUseSearchProfiles).toHaveBeenCalledWith("agency")
  })

  it("renders correct number of cards for results", () => {
    const profiles = [
      createProfile({ user_id: "1", first_name: "A", last_name: "One" }),
      createProfile({ user_id: "2", first_name: "B", last_name: "Two" }),
      createProfile({ user_id: "3", first_name: "C", last_name: "Three" }),
    ]

    mockUseSearchProfiles.mockReturnValue(mockInfiniteResult(profiles))

    renderSearchPage("freelancer")

    const links = screen.getAllByRole("link")
    expect(links).toHaveLength(3)
  })

  it("shows load more button when hasNextPage is true", () => {
    const profiles = [
      createProfile({ user_id: "1", first_name: "A", last_name: "One" }),
    ]

    mockUseSearchProfiles.mockReturnValue(
      mockInfiniteResult(profiles, { hasNextPage: true }),
    )

    renderSearchPage("freelancer")

    expect(
      screen.getByRole("button", { name: messages.search.loadMore }),
    ).toBeInTheDocument()
  })
})
