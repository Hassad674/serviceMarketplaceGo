/**
 * referrers/[id]/page generateMetadata golden tests — PERF-W-08.
 */

import { describe, it, expect, vi, beforeEach } from "vitest"

const fetchMock = vi.fn()
vi.mock("@/features/referrer-profile/api/referrer-profile-server", () => ({
  fetchReferrerProfileForMetadata: (...args: unknown[]) => fetchMock(...args),
}))

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
  const mod = await import("../referrers/[id]/page")
  return (await mod.generateMetadata({
    params: Promise.resolve({ id, locale }),
  })) as Record<string, unknown>
}

describe("referrers/[id] generateMetadata — PERF-W-08", () => {
  it("emits canonical + hreflang on a populated profile", async () => {
    fetchMock.mockResolvedValue({
      organization_id: "ref-1",
      title: "Top apporteur",
      about: "Connecte les bonnes équipes.",
      photo_url: "",
      city: "Paris",
      country_code: "FR",
      languages_professional: ["fr"],
    })
    const md = await callMetadata("ref-1", "fr")
    const alternates = md.alternates as {
      canonical: string
      languages: Record<string, string>
    }
    expect(alternates.canonical).toMatch(/\/fr\/referrers\/ref-1$/)
    expect(alternates.languages.en).toMatch(/\/en\/referrers\/ref-1$/)
    expect(alternates.languages["x-default"]).toMatch(
      /\/fr\/referrers\/ref-1$/,
    )
  })

  it("falls back gracefully when the fetcher returns null", async () => {
    fetchMock.mockResolvedValue(null)
    const md = await callMetadata("ref-x", "en")
    const alternates = md.alternates as { canonical: string }
    expect(alternates.canonical).toMatch(/\/en\/referrers\/ref-x$/)
  })
})
