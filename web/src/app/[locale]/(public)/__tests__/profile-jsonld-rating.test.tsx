/**
 * profile-jsonld-rating.test.tsx — guards the contract that profile
 * JSON-LD payloads NEVER emit `aggregateRating` or `review` when the
 * entity has zero reviews. Google rejects rich-result eligibility for
 * pages with hollow rating blocks.
 *
 * The test renders the page component (Server Component) and reads the
 * generated <script type="application/ld+json"> payload to assert the
 * structure. This is the strongest possible guard: any future regression
 * that re-introduces an empty AggregateRating fails this test.
 */

import { describe, it, expect, vi, beforeEach } from "vitest"
import { renderToStaticMarkup } from "react-dom/server"
import React from "react"

// --- mocks ---------------------------------------------------------------

const fetchProfileMock = vi.fn()
vi.mock("@/features/freelance-profile/api/freelance-profile-server", () => ({
  fetchFreelanceProfileForMetadata: (...args: unknown[]) =>
    fetchProfileMock(...args),
}))

const fetchAvgMock = vi.fn()
const fetchReviewsMock = vi.fn()
const fetchRelatedMock = vi.fn()
vi.mock("@/shared/lib/seo/server-fetchers", () => ({
  fetchPublicAverageRating: (...args: unknown[]) => fetchAvgMock(...args),
  fetchPublicReviews: (...args: unknown[]) => fetchReviewsMock(...args),
  fetchRelatedProfiles: (...args: unknown[]) => fetchRelatedMock(...args),
}))

vi.mock("@/features/messaging/components/send-message-button", () => ({
  SendMessageButton: () => null,
}))

vi.mock(
  "@/features/freelance-profile/components/freelance-public-profile-loader",
  () => ({
    FreelancePublicProfileLoader: () => null,
  }),
)

vi.mock("@i18n/navigation", () => ({
  Link: ({
    children,
    ...rest
  }: {
    children: React.ReactNode
    [key: string]: unknown
  }) =>
    React.createElement(
      "a",
      rest as React.AnchorHTMLAttributes<HTMLAnchorElement>,
      children,
    ),
}))

vi.mock("next-intl/server", () => ({
  getTranslations: async ({ namespace }: { namespace: string }) => {
    return (key: string) => `${namespace}.${key}`
  },
}))

beforeEach(() => {
  fetchProfileMock.mockReset()
  fetchAvgMock.mockReset()
  fetchReviewsMock.mockReset()
  fetchRelatedMock.mockReset()
  fetchRelatedMock.mockResolvedValue([])
})

async function renderPage(id: string, locale: string): Promise<string> {
  const mod = await import("../freelancers/[id]/page")
  const element = await mod.default({
    params: Promise.resolve({ id, locale }),
  })
  return renderToStaticMarkup(element as React.ReactElement)
}

function extractFirstJsonLd(html: string): Record<string, unknown> {
  const match = html.match(
    /<script type="application\/ld\+json">([\s\S]*?)<\/script>/,
  )
  if (!match) throw new Error("no JSON-LD script tag found")
  return JSON.parse(match[1])
}

describe("freelancer JSON-LD aggregate rating guard", () => {
  it("OMITS aggregateRating + review when the org has zero reviews", async () => {
    fetchProfileMock.mockResolvedValue({
      organization_id: "free-1",
      title: "Designer",
      about: "About",
      photo_url: "",
      city: "Paris",
      country_code: "FR",
      skills: [],
      expertise_domains: [],
      languages_professional: [],
    })
    fetchAvgMock.mockResolvedValue({ average: 0, count: 0 })
    fetchReviewsMock.mockResolvedValue([])

    const html = await renderPage("free-1", "fr")
    const payload = extractFirstJsonLd(html)
    expect(payload["@type"]).toBe("Person")
    // The schema must NOT include either field when the org has no
    // reviews — Google's Rich Results Test fails the page otherwise.
    expect(payload.aggregateRating).toBeUndefined()
    expect(payload.review).toBeUndefined()
  })

  it("emits aggregateRating + review[] when the org has ≥ 1 review", async () => {
    fetchProfileMock.mockResolvedValue({
      organization_id: "free-2",
      title: "Designer",
      about: "About",
      photo_url: "",
      city: "Paris",
      country_code: "FR",
      skills: [{ skill_text: "design", display_text: "Design" }],
      expertise_domains: ["web"],
      languages_professional: [],
    })
    fetchAvgMock.mockResolvedValue({ average: 4.8, count: 3 })
    fetchReviewsMock.mockResolvedValue([
      {
        id: "r-1",
        proposal_id: "p-1",
        reviewer_id: "u-1",
        reviewed_id: "u-2",
        global_rating: 5,
        timeliness: 5,
        communication: 5,
        quality: 5,
        comment: "Great",
        video_url: null,
        title_visible: false,
        side: "client_to_provider",
        published_at: "2026-04-01T00:00:00Z",
        created_at: "2026-04-01T00:00:00Z",
      },
    ])

    const html = await renderPage("free-2", "fr")
    const payload = extractFirstJsonLd(html)
    expect(payload.aggregateRating).toMatchObject({
      "@type": "AggregateRating",
      reviewCount: 3,
      bestRating: 5,
    })
    const reviews = payload.review as unknown[]
    expect(reviews).toHaveLength(1)
  })
})
