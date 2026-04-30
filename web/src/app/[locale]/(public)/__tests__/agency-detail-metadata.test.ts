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

describe("agencies/[id] generateMetadata — PERF-W-06", () => {
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
    expect((md.alternates as { canonical: string }).canonical).toBe(
      "/agencies/agency-1",
    )
    const og = md.openGraph as { images?: Array<{ url: string }> }
    expect(og.images).toBeDefined()
    expect(og.images?.[0]?.url).toBe("https://r2/logo.jpg")
  })

  it("falls back to generic title when profile is missing", async () => {
    fetchMock.mockResolvedValue(null)
    const md = await callMetadata("agency-x")
    expect(md.title as string).toContain("publicProfile.agencyProfile")
    expect(md.description as string).toContain("publicProfile.agencyProfileDesc")
    expect((md.alternates as { canonical: string }).canonical).toBe(
      "/agencies/agency-x",
    )
    const og = md.openGraph as { images?: unknown }
    expect(og.images).toBeUndefined()
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
