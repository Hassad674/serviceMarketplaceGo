import { describe, expect, it, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import { NextIntlClientProvider } from "next-intl"
import messages from "@/../messages/en.json"
import { SearchResultCard } from "../search-result-card"
import type { SearchDocument } from "@/shared/lib/search/search-document"

// ---- Test doubles for Next.js integrations ----

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

// ---- Fixtures ----

function makeDocument(overrides: Partial<SearchDocument> = {}): SearchDocument {
  return {
    id: "org-1",
    persona: "freelance",
    display_name: "Alice Martin",
    title: "Go Backend Engineer",
    photo_url: "",
    city: "Paris",
    country_code: "FR",
    languages_professional: ["fr", "en"],
    availability_status: "available_now",
    expertise_domains: [],
    skills: ["Go", "TypeScript", "React", "Kubernetes"],
    pricing: {
      type: "daily",
      min_amount: 60000,
      max_amount: null,
      currency: "EUR",
      negotiable: true,
    },
    rating: { average: 4.8, count: 12 },
    total_earned: 1234500,
    completed_projects: 24,
    created_at: "2026-02-01T00:00:00Z",
    ...overrides,
  }
}

function renderCard(doc: SearchDocument) {
  return render(
    <NextIntlClientProvider locale="en" messages={messages}>
      <SearchResultCard document={doc} />
    </NextIntlClientProvider>,
  )
}

// ---- Tests ----

describe("SearchResultCard", () => {
  it("renders the display name and title", () => {
    renderCard(makeDocument())
    expect(screen.getByText("Alice Martin")).toBeInTheDocument()
    expect(screen.getByText("Go Backend Engineer")).toBeInTheDocument()
  })

  it("links to the freelancers detail page for a freelance persona", () => {
    renderCard(makeDocument({ persona: "freelance", id: "abc" }))
    const link = screen.getByRole("link")
    expect(link.getAttribute("href")).toBe("/freelancers/abc")
  })

  it("links to the agencies detail page for an agency persona", () => {
    renderCard(makeDocument({ persona: "agency", id: "acme" }))
    const link = screen.getByRole("link")
    expect(link.getAttribute("href")).toBe("/agencies/acme")
  })

  it("links to the referrers detail page for a referrer persona", () => {
    renderCard(makeDocument({ persona: "referrer", id: "rita" }))
    const link = screen.getByRole("link")
    expect(link.getAttribute("href")).toBe("/referrers/rita")
  })

  it("shows the rating badge only when there is at least one review", () => {
    renderCard(makeDocument({ rating: { average: 4.8, count: 12 } }))
    expect(screen.getByText("4.8")).toBeInTheDocument()
  })

  it("hides the rating badge when review count is zero", () => {
    renderCard(
      makeDocument({
        rating: { average: 0, count: 0 },
        total_earned: 0,
      }),
    )
    expect(screen.queryByText(/4\.8/)).not.toBeInTheDocument()
  })

  it("renders the total-earned line when amount > 0", () => {
    renderCard(makeDocument({ total_earned: 1234500 }))
    // Match the formatted amount (en-US: "€12,345") appearing inside
    // the "earned" line — we only assert the digits to stay resilient
    // against Intl whitespace quirks.
    expect(screen.getByText(/12,345/)).toBeInTheDocument()
  })

  it("hides the total-earned line at zero", () => {
    const { container } = renderCard(makeDocument({ total_earned: 0 }))
    // The total-earned line is the only primary-color paragraph. None
    // should be present when the amount is zero.
    expect(
      container.querySelectorAll("p.text-rose-600"),
    ).toHaveLength(0)
  })

  it("renders the pricing line with a negotiable badge when applicable", () => {
    renderCard(
      makeDocument({
        pricing: {
          type: "daily",
          min_amount: 60000,
          max_amount: null,
          currency: "EUR",
          negotiable: true,
        },
      }),
    )
    expect(screen.getByText(messages.search.negotiable)).toBeInTheDocument()
  })

  it("hides the pricing line when pricing is null", () => {
    renderCard(makeDocument({ pricing: null }))
    expect(
      screen.queryByText(messages.search.negotiable),
    ).not.toBeInTheDocument()
  })

  it("limits skill chips to 3 visible with a +N overflow", () => {
    renderCard(
      makeDocument({ skills: ["Go", "TypeScript", "React", "Kubernetes"] }),
    )
    expect(screen.getByText("Go")).toBeInTheDocument()
    expect(screen.getByText("TypeScript")).toBeInTheDocument()
    expect(screen.getByText("React")).toBeInTheDocument()
    expect(screen.queryByText("Kubernetes")).not.toBeInTheDocument()
    expect(screen.getByText(/\+1/)).toBeInTheDocument()
  })

  it("falls back to initials when no photo is provided", () => {
    renderCard(makeDocument({ photo_url: "" }))
    expect(screen.getByText("AM")).toBeInTheDocument()
  })

  it("uses a semantic article wrapper for accessibility", () => {
    renderCard(makeDocument())
    expect(screen.getByRole("article")).toBeInTheDocument()
  })
})
