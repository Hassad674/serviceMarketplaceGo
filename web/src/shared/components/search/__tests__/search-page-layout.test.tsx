import { describe, expect, it, vi } from "vitest"
import { fireEvent, render, screen } from "@testing-library/react"
import { NextIntlClientProvider } from "next-intl"
import messages from "@/../messages/en.json"
import { SearchPageLayout } from "../search-page-layout"
import type { RawSearchDocumentLike } from "@/shared/lib/search/search-document-adapter"

vi.mock("@i18n/navigation", () => ({
  Link: ({
    href,
    children,
    className,
  }: {
    href: string
    children: React.ReactNode
    className?: string
  }) => (
    <a href={href} className={className}>
      {children}
    </a>
  ),
}))

vi.mock("next/image", () => ({
  default: ({ src, alt }: { src: string; alt: string }) => (
    // eslint-disable-next-line @next/next/no-img-element
    <img src={src} alt={alt} />
  ),
}))

function renderLayout(
  props: Partial<React.ComponentProps<typeof SearchPageLayout>> = {},
) {
  const defaults: React.ComponentProps<typeof SearchPageLayout> = {
    title: "Find freelancers",
    persona: "freelance",
    documents: [],
    status: "idle",
    hasMore: false,
    isLoadingMore: false,
    onLoadMore: vi.fn(),
    query: "",
    onQueryChange: vi.fn(),
  }
  return render(
    <NextIntlClientProvider locale="en" messages={messages}>
      <SearchPageLayout {...defaults} {...props} />
    </NextIntlClientProvider>,
  )
}

const sampleDoc: RawSearchDocumentLike = {
  organization_id: "org-1",
  name: "Alice",
  title: "Senior Go Dev",
  photo_url: "",
  city: "Paris",
  country_code: "FR",
  languages_professional: ["fr", "en"],
  availability_status: "available_now",
  total_earned: 1000000,
  completed_projects: 10,
  skills: [
    { display_text: "Go", skill_text: "go" },
    { display_text: "React", skill_text: "react" },
  ],
  average_rating: 4.9,
  review_count: 8,
}

describe("SearchPageLayout", () => {
  it("renders the page title", () => {
    renderLayout({ title: "Find freelancers" })
    expect(
      screen.getByRole("heading", { level: 1, name: "Find freelancers" }),
    ).toBeInTheDocument()
  })

  it("shows skeleton placeholders when loading", () => {
    const { container } = renderLayout({ status: "loading" })
    expect(container.querySelectorAll(".animate-shimmer").length).toBeGreaterThan(
      0,
    )
  })

  it("shows the empty state when there are no documents", () => {
    renderLayout({ status: "idle", documents: [] })
    // Title ("No results") also appears in the sidebar resultsCount
    // plural, so assert on the unique empty-state description instead.
    expect(
      screen.getByText(messages.search.empty.description),
    ).toBeInTheDocument()
    expect(
      screen.getByRole("button", { name: messages.search.empty.cta }),
    ).toBeInTheDocument()
  })

  it("renders a card per document", () => {
    renderLayout({ documents: [sampleDoc] })
    expect(screen.getByText("Alice")).toBeInTheDocument()
    expect(screen.getByText("Senior Go Dev")).toBeInTheDocument()
  })

  it("shows the load-more button when hasMore is true", () => {
    renderLayout({ documents: [sampleDoc], hasMore: true })
    expect(
      screen.getByRole("button", { name: messages.search.loadMore }),
    ).toBeInTheDocument()
  })

  it("fires onQueryChange when the search input changes", () => {
    const onQueryChange = vi.fn()
    renderLayout({ onQueryChange })
    fireEvent.change(
      screen.getByPlaceholderText(messages.search.searchPlaceholder),
      { target: { value: "go" } },
    )
    expect(onQueryChange).toHaveBeenCalledWith("go")
  })

  it("shows the error state when status is error", () => {
    renderLayout({ status: "error" })
    expect(screen.getByText(messages.search.errorLoading)).toBeInTheDocument()
  })
})
