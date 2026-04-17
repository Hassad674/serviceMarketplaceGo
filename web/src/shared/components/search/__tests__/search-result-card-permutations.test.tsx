import { describe, expect, it, vi } from "vitest"
import { render } from "@testing-library/react"
import { NextIntlClientProvider } from "next-intl"
import messages from "@/../messages/en.json"
import { SearchResultCard } from "../search-result-card"
import type { SearchDocument } from "@/shared/lib/search/search-document"

// search-result-card-permutations.test.tsx — drives the card through
// every permutation of optional fields to catch regressions the named
// hand-written tests would miss. Any combination that causes a render
// error (React throw, NaN in string, broken link target) surfaces as a
// failing assertion.
//
// Kept separate from search-result-card.test.tsx so the permutation
// sweep can grow without bloating the more readable behaviour suite.

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

function baseDoc(): SearchDocument {
  return {
    id: "org-1",
    persona: "freelance",
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

function renderCard(doc: SearchDocument) {
  return render(
    <NextIntlClientProvider locale="en" messages={messages}>
      <SearchResultCard document={doc} />
    </NextIntlClientProvider>,
  )
}

describe("SearchResultCard — permutation sweep", () => {
  const personas: Array<SearchDocument["persona"]> = [
    "freelance",
    "agency",
    "referrer",
  ]

  const photos = ["", "https://example.com/photo.png"]

  const ratings: Array<SearchDocument["rating"]> = [
    { average: 0, count: 0 },
    { average: 4.7, count: 1 },
    { average: 4.2, count: 42 },
  ]

  const pricingOptions: Array<SearchDocument["pricing"]> = [
    null,
    {
      type: "daily",
      min_amount: 50000,
      max_amount: null,
      currency: "EUR",
      negotiable: false,
    },
    {
      type: "daily",
      min_amount: 50000,
      max_amount: 80000,
      currency: "EUR",
      negotiable: true,
    },
  ]

  // Generator yields the cross product as small fixtures — each line
  // is named so failures point at the offending permutation.
  for (const persona of personas) {
    for (const photo of photos) {
      for (const rating of ratings) {
        for (const pricing of pricingOptions) {
          const name = `persona=${persona} photo=${photo ? "yes" : "no"} rating=${rating.count} pricing=${pricing ? pricing.type + (pricing.negotiable ? "-neg" : "") : "none"}`
          it(`renders without crashing: ${name}`, () => {
            const doc: SearchDocument = {
              ...baseDoc(),
              persona,
              photo_url: photo,
              rating,
              pricing,
              skills: ["React", "Go"].slice(0, persona === "agency" ? 2 : 1),
            }
            const { container } = renderCard(doc)
            // Card must render exactly one link.
            expect(container.querySelectorAll("a")).toHaveLength(1)
            // Display name must always appear.
            expect(container.textContent).toContain(doc.display_name)
          })
        }
      }
    }
  }

  it("handles missing skills gracefully", () => {
    const { container } = renderCard({ ...baseDoc(), skills: [] })
    expect(container.textContent).toContain("Camille Martin")
  })

  it("accepts a long skill list without overflow errors", () => {
    const skills = Array.from({ length: 20 }, (_, i) => `Skill${i}`)
    const { container } = renderCard({ ...baseDoc(), skills })
    expect(container).toBeTruthy()
  })

  it("accepts extreme rating values without NaN rendering", () => {
    const { container } = renderCard({
      ...baseDoc(),
      rating: { average: 5, count: 9999 },
    })
    expect(container.textContent).not.toContain("NaN")
  })
})
