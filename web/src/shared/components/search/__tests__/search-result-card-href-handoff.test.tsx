import { describe, expect, it, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import { NextIntlClientProvider } from "next-intl"
import messages from "@/../messages/en.json"
import {
  SearchResultCard,
  buildResultHref,
} from "../search-result-card"
import type { SearchDocument } from "@/shared/lib/search/search-document"

// Search handoff regression suite — locks in the contract that
// clicking a search result navigates to the destination profile with
// `?q=<lowercased trimmed query>&pos=<1-based rank>` query params.
// The backend's tracking middleware reads those params on the public
// profile GET and emits a profile_view row with `search_query` and
// `search_position` populated. Without them, /me/stats/keywords stays
// empty and average position degrades to NULL — a silent regression
// that the unit tests would otherwise miss.

vi.mock("@i18n/navigation", () => ({
  Link: ({ href, children }: { href: string; children: React.ReactNode }) => (
    <a href={href} data-testid="result-link">
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

function baseDoc(persona: SearchDocument["persona"] = "freelance"): SearchDocument {
  return {
    id: "org-42",
    persona,
    display_name: "Camille Martin",
    title: "Senior React Developer",
    photo_url: "",
    city: "Paris",
    country_code: "FR",
    languages_professional: ["fr"],
    availability_status: "available_now",
    expertise_domains: [],
    skills: [],
    pricing: null,
    rating: { average: 0, count: 0 },
    total_earned: 0,
    completed_projects: 0,
    created_at: "2026-02-01T00:00:00Z",
  }
}

function renderCard(props: {
  query?: string
  position?: number
  persona?: SearchDocument["persona"]
}) {
  return render(
    <NextIntlClientProvider locale="en" messages={messages}>
      <SearchResultCard
        document={baseDoc(props.persona)}
        query={props.query}
        position={props.position}
      />
    </NextIntlClientProvider>,
  )
}

describe("buildResultHref", () => {
  it("returns the bare href when no query / position", () => {
    expect(buildResultHref("freelance", "abc", undefined, undefined)).toBe(
      "/freelancers/abc",
    )
  })

  it("appends q + pos when both are provided", () => {
    expect(buildResultHref("agency", "id", "Go Developer", 3)).toBe(
      "/agencies/id?q=go%20developer&pos=3",
    )
  })

  it("lowercases and trims the query", () => {
    expect(buildResultHref("freelance", "id", "  React DEV  ", 1)).toBe(
      "/freelancers/id?q=react%20dev&pos=1",
    )
  })

  it("URL-encodes special characters", () => {
    expect(buildResultHref("freelance", "id", "node.js & express", 2)).toBe(
      "/freelancers/id?q=node.js%20%26%20express&pos=2",
    )
  })

  it("omits q when the trimmed query is empty", () => {
    expect(buildResultHref("freelance", "id", "   ", 5)).toBe(
      "/freelancers/id?pos=5",
    )
  })

  it("omits pos when the value is below 1 or non-finite", () => {
    expect(buildResultHref("freelance", "id", "react", 0)).toBe(
      "/freelancers/id?q=react",
    )
    expect(buildResultHref("freelance", "id", "react", Number.NaN)).toBe(
      "/freelancers/id?q=react",
    )
    expect(buildResultHref("freelance", "id", "react", Number.POSITIVE_INFINITY)).toBe(
      "/freelancers/id?q=react",
    )
  })

  it("truncates fractional positions to integers", () => {
    expect(buildResultHref("freelance", "id", "react", 3.7)).toBe(
      "/freelancers/id?q=react&pos=3",
    )
  })

  it("maps every persona to the right path prefix", () => {
    expect(buildResultHref("freelance", "id", undefined, undefined)).toMatch(
      /^\/freelancers\//,
    )
    expect(buildResultHref("agency", "id", undefined, undefined)).toMatch(
      /^\/agencies\//,
    )
    expect(buildResultHref("referrer", "id", undefined, undefined)).toMatch(
      /^\/referrers\//,
    )
  })
})

describe("SearchResultCard — handoff href", () => {
  it("appends ?q=&pos= when the layout passes both", () => {
    renderCard({ query: "react developer", position: 3 })
    const link = screen.getByTestId("result-link") as HTMLAnchorElement
    expect(link.getAttribute("href")).toBe(
      "/freelancers/org-42?q=react%20developer&pos=3",
    )
  })

  it("strips empty query but keeps position", () => {
    renderCard({ query: "", position: 7 })
    expect(
      (screen.getByTestId("result-link") as HTMLAnchorElement).getAttribute("href"),
    ).toBe("/freelancers/org-42?pos=7")
  })

  it("works without any handoff params (legacy SQL path)", () => {
    renderCard({})
    expect(
      (screen.getByTestId("result-link") as HTMLAnchorElement).getAttribute("href"),
    ).toBe("/freelancers/org-42")
  })

  it("respects the persona prefix", () => {
    renderCard({ persona: "agency", query: "design", position: 1 })
    expect(
      (screen.getByTestId("result-link") as HTMLAnchorElement).getAttribute("href"),
    ).toBe("/agencies/org-42?q=design&pos=1")
  })
})
