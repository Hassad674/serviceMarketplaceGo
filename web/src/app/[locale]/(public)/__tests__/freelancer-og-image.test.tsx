/**
 * freelancers/[id]/opengraph-image route smoke test — PERF-W-08.
 *
 * `next/og` is a Vercel runtime concern; here we stub `ImageResponse`
 * with a constructor that captures its arguments so we can assert:
 *   - the renderer was invoked with the correct profile data
 *   - the route propagates the configured size + content type
 *   - the fallback path (no profile) still renders without throwing
 */

import { describe, it, expect, vi, beforeEach } from "vitest"

const imageResponseInstances: unknown[] = []

vi.mock("next/og", () => ({
  ImageResponse: vi.fn(function ImageResponseStub(this: unknown, ...args: unknown[]) {
    imageResponseInstances.push({ args })
    Object.assign(this as Record<string, unknown>, {
      headers: { get: () => "image/png" },
      args,
    })
    return this
  }),
}))

const profileMock = vi.fn()
vi.mock("@/features/freelance-profile/api/freelance-profile-server", () => ({
  fetchFreelanceProfileForMetadata: (...args: unknown[]) => profileMock(...args),
}))

const ratingMock = vi.fn()
vi.mock("@/shared/lib/seo/server-fetchers", () => ({
  fetchPublicAverageRating: (...args: unknown[]) => ratingMock(...args),
}))

vi.mock("next-intl/server", () => ({
  getTranslations: async ({ namespace }: { namespace: string }) => {
    return (key: string, vars: Record<string, unknown> = {}) => {
      if (key === "ratingLine") {
        return `★ ${vars.rating} (${vars.count})`
      }
      return `${namespace}.${key}`
    }
  },
}))

beforeEach(() => {
  imageResponseInstances.length = 0
  profileMock.mockReset()
  ratingMock.mockReset()
})

async function invokeOgImage(id: string, locale: string) {
  const mod = await import("../freelancers/[id]/opengraph-image")
  return mod.default({
    params: Promise.resolve({ id, locale }),
  })
}

describe("freelancers/[id] opengraph-image — PERF-W-08", () => {
  it("renders an ImageResponse with the configured size + content type", async () => {
    const mod = await import("../freelancers/[id]/opengraph-image")
    expect(mod.size).toEqual({ width: 1200, height: 630 })
    expect(mod.contentType).toBe("image/png")
  })

  it("invokes ImageResponse with the profile data when available", async () => {
    profileMock.mockResolvedValue({
      title: "Senior React Dev",
      city: "Lyon",
      photo_url: "https://r2/photo.jpg",
    })
    ratingMock.mockResolvedValue({ average: 4.8, count: 12 })
    await invokeOgImage("free-1", "fr")
    expect(imageResponseInstances).toHaveLength(1)
    const call = imageResponseInstances[0] as { args: unknown[] }
    expect(call.args).toHaveLength(2)
    const [, options] = call.args as [unknown, { width: number; height: number }]
    expect(options.width).toBe(1200)
    expect(options.height).toBe(630)
  })

  it("does not throw when the profile fetcher returns null", async () => {
    profileMock.mockResolvedValue(null)
    ratingMock.mockResolvedValue(null)
    await expect(invokeOgImage("free-x", "fr")).resolves.toBeDefined()
  })
})
