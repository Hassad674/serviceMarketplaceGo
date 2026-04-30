import { describe, it, expect, vi, afterEach } from "vitest"

vi.mock("@/shared/lib/api-client", () => ({
  API_BASE_URL: "http://localhost:8080",
}))

import { fetchAgencyProfileForMetadata } from "../agency-profile-server"

afterEach(() => {
  vi.restoreAllMocks()
})

describe("fetchAgencyProfileForMetadata — PERF-W-06", () => {
  it("hits /api/v1/profiles/:id with ISR revalidate", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({
        organization_id: "agency-1",
        title: "Acme Agency",
        about: "We do things",
        photo_url: "",
        presentation_video_url: "",
        referrer_video_url: "",
        referrer_about: "",
        created_at: "",
        updated_at: "",
      }),
    })
    vi.stubGlobal("fetch", fetchMock)

    const result = await fetchAgencyProfileForMetadata("agency-1")
    expect(fetchMock.mock.calls[0][0]).toBe(
      "http://localhost:8080/api/v1/profiles/agency-1",
    )
    const init = fetchMock.mock.calls[0][1] as { next?: { revalidate?: number } }
    expect(init.next?.revalidate).toBe(120)
    expect(result?.title).toBe("Acme Agency")
  })

  it("returns null on non-200 (preserves crawl budget)", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({ ok: false, json: async () => ({}) }),
    )
    const result = await fetchAgencyProfileForMetadata("agency-x")
    expect(result).toBeNull()
  })

  it("returns null when fetch throws", async () => {
    vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new Error("net")))
    const result = await fetchAgencyProfileForMetadata("agency-x")
    expect(result).toBeNull()
  })
})
