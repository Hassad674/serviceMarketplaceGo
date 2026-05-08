/**
 * agencies/[id]/page generateMetadata tests — PERF-W-06.
 *
 * Asserts:
 *   - title interpolates the agency display name
 *   - canonical URL uses the agency id
 *   - description falls back to a generic string when the profile
 *     has no `about`
 *   - openGraph image uses photo_url when present
 *   - graceful fallback when the fetcher returns null
 */

import { describe, it, expect, vi, beforeEach } from "vitest"

const fetchMock = vi.fn()
vi.mock("@/features/provider/api/agency-profile-server", () => ({
  fetchAgencyProfileForMetadata: (...args: unknown[]) => fetchMock(...args),
}))

vi.mock("@/features/provider/components/public-profile", () => ({
  PublicProfile: () => null,
}))

vi.mock("@/features/messaging/components/send-message-button", () => ({
  SendMessageButton: () => null,
}))

// `@i18n/navigation` pulls in `next/navigation` which is unavailable
// in the vitest jsdom environment — stub the Link export with a plain
// anchor so the page module imports cleanly when only metadata is
// being exercised.
vi.mock("@i18n/navigation", () => ({
  Link: () => null,
}))

vi.mock("next-intl/server", () => ({
  getTranslations: async ({ namespace }: { namespace: string }) => {
    return (key: string) => `${namespace}.${key}`
  },
}))

beforeEach(() => {
  fetchMock.mockReset()
})

async function callMetadata(id: string): Promise<Record<string, unknown>> {
  const mod = await import("../agencies/[id]/page")
  return (await mod.generateMetadata({
    params: Promise.resolve({ id, locale: "en" }),
  })) as Record<string, unknown>
}

describe("agencies/[id] generateMetadata — PERF-W-06 + PERF-W-08", () => {
  it("interpolates agency title and about into title/description", async () => {
    fetchMock.mockResolvedValue({
      organization_id: "agency-1",
      title: "Acme Agency",
      about: "Crafting beautiful B2B websites since 2018.",
      photo_url: "https://r2/logo.jpg",
      city: "Paris",
      country_code: "FR",
    })

    const md = await callMetadata("agency-1")
    expect(md.title as string).toContain("Acme Agency")
    expect(md.title as string).toContain("publicProfile.agencyProfile")
    expect(md.description as string).toBe(
      "Crafting beautiful B2B websites since 2018.",
    )
    const alternates = md.alternates as {
      canonical: string
      languages: Record<string, string>
    }
    // Locale-aware absolute canonical URL — keeps each indexed
    // version mapped to its own language entry.
    expect(alternates.canonical).toMatch(/\/en\/agencies\/agency-1$/)
    // hreflang map must declare both locales + x-default for Google.
    expect(alternates.languages.fr).toMatch(/\/fr\/agencies\/agency-1$/)
    expect(alternates.languages.en).toMatch(/\/en\/agencies\/agency-1$/)
    expect(alternates.languages["x-default"]).toMatch(
      /\/fr\/agencies\/agency-1$/,
    )
    // PERF-W-08: OG image is now served by the colocated
    // opengraph-image route, so we no longer set `images` here —
    // Next.js auto-wires the route into the metadata at build time.
    const og = md.openGraph as { type?: string; locale?: string }
    expect(og.type).toBe("profile")
    expect(og.locale).toBe("en_US")
    const twitter = md.twitter as { card?: string }
    expect(twitter.card).toBe("summary_large_image")
  })

  it("falls back to generic title when profile is missing", async () => {
    fetchMock.mockResolvedValue(null)
    const md = await callMetadata("agency-x")
    expect(md.title as string).toContain("publicProfile.agencyProfile")
    expect(md.description as string).toContain("publicProfile.agencyProfileDesc")
    const alternates = md.alternates as {
      canonical: string
      languages: Record<string, string>
    }
    expect(alternates.canonical).toMatch(/\/en\/agencies\/agency-x$/)
    expect(alternates.languages.fr).toMatch(/\/fr\/agencies\/agency-x$/)
  })

  it("truncates long about strings to 160 chars for description", async () => {
    const longAbout = "x".repeat(500)
    fetchMock.mockResolvedValue({
      organization_id: "agency-2",
      title: "Big Agency",
      about: longAbout,
      photo_url: "",
      city: "",
      country_code: "",
    })

    const md = await callMetadata("agency-2")
    expect((md.description as string).length).toBe(160)
  })
})
