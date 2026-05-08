/**
 * freelancers/[id]/page generateMetadata golden tests — PERF-W-08.
 *
 * Pins the metadata contract for the freelance profile detail page:
 *   - title interpolates the freelance display name
 *   - canonical is the absolute, locale-prefixed URL
 *   - hreflang declares both supported locales + x-default
 *   - openGraph type is "profile" with the absolute canonical URL
 *   - twitter card is "summary_large_image" so the OG image surfaces
 *     in tweets / link previews
 */

import { describe, it, expect, vi, beforeEach } from "vitest"

const fetchMock = vi.fn()
vi.mock("@/features/freelance-profile/api/freelance-profile-server", () => ({
  fetchFreelanceProfileForMetadata: (...args: unknown[]) => fetchMock(...args),
}))

// Stub the locale-aware Link — `next/navigation` is not loadable in
// the vitest jsdom env. The page only references it for the back link.
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

async function callMetadata(
  id: string,
  locale: string,
): Promise<Record<string, unknown>> {
  const mod = await import("../freelancers/[id]/page")
  return (await mod.generateMetadata({
    params: Promise.resolve({ id, locale }),
  })) as Record<string, unknown>
}

describe("freelancers/[id] generateMetadata — PERF-W-08", () => {
  it("interpolates freelance title and emits hreflang for fr", async () => {
    fetchMock.mockResolvedValue({
      organization_id: "free-1",
      title: "Senior React Developer",
      about: "Crafting beautiful interfaces.",
      photo_url: "https://r2/photo.jpg",
      city: "Lyon",
      country_code: "FR",
      skills: [{ skill_text: "react", display_text: "React" }],
      languages_professional: ["fr", "en"],
    })

    const md = await callMetadata("free-1", "fr")
    expect(md.title as string).toContain("Senior React Developer")
    const alternates = md.alternates as {
      canonical: string
      languages: Record<string, string>
    }
    expect(alternates.canonical).toMatch(/\/fr\/freelancers\/free-1$/)
    expect(alternates.languages.fr).toMatch(/\/fr\/freelancers\/free-1$/)
    expect(alternates.languages.en).toMatch(/\/en\/freelancers\/free-1$/)
    expect(alternates.languages["x-default"]).toMatch(
      /\/fr\/freelancers\/free-1$/,
    )
    const og = md.openGraph as {
      type?: string
      url?: string
      locale?: string
    }
    expect(og.type).toBe("profile")
    expect(og.url).toMatch(/\/fr\/freelancers\/free-1$/)
    expect(og.locale).toBe("fr_FR")
    const twitter = md.twitter as { card?: string }
    expect(twitter.card).toBe("summary_large_image")
  })

  it("emits the correct canonical for the en locale", async () => {
    fetchMock.mockResolvedValue({
      organization_id: "free-2",
      title: "Designer",
      about: "",
      photo_url: "",
      city: "",
      country_code: "",
      skills: [],
      languages_professional: [],
    })
    const md = await callMetadata("free-2", "en")
    const alternates = md.alternates as { canonical: string }
    expect(alternates.canonical).toMatch(/\/en\/freelancers\/free-2$/)
    const og = md.openGraph as { locale?: string }
    expect(og.locale).toBe("en_US")
  })

  it("falls back to generic title when profile fetch fails", async () => {
    fetchMock.mockResolvedValue(null)
    const md = await callMetadata("free-x", "fr")
    expect(md.title as string).toContain("profile.freelance.publicTitleSuffix")
    const alternates = md.alternates as { canonical: string }
    expect(alternates.canonical).toMatch(/\/fr\/freelancers\/free-x$/)
  })
})
