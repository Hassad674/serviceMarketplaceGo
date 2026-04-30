import { describe, it, expect, vi, afterEach } from "vitest"

vi.mock("@/shared/lib/api-client", () => ({
  API_BASE_URL: "http://localhost:8080",
}))

import { fetchSitemapJobs } from "../sitemap-server"

afterEach(() => {
  vi.restoreAllMocks()
})

describe("fetchSitemapJobs — PERF-W-04 sitemap source", () => {
  it("hits /api/v1/jobs?status=open with ISR", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({
        data: [
          { id: "job-1", updated_at: "2026-04-30T00:00:00Z" },
          { id: "job-2", updated_at: "2026-04-29T00:00:00Z" },
        ],
      }),
    })
    vi.stubGlobal("fetch", fetchMock)

    const result = await fetchSitemapJobs()
    const url = fetchMock.mock.calls[0][0] as string
    expect(url).toContain("/api/v1/jobs")
    expect(url).toContain("status=open")
    expect(url).toContain("per_page=200")
    expect(result).toHaveLength(2)
    expect(result[0].id).toBe("job-1")
  })

  it("falls back to .jobs envelope when .data is absent", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        json: async () => ({
          jobs: [{ id: "job-3", updated_at: "" }],
        }),
      }),
    )
    const result = await fetchSitemapJobs()
    expect(result[0].id).toBe("job-3")
  })

  it("returns [] on non-200", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({ ok: false, json: async () => ({}) }),
    )
    expect(await fetchSitemapJobs()).toEqual([])
  })

  it("returns [] on network failure", async () => {
    vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new Error("net")))
    expect(await fetchSitemapJobs()).toEqual([])
  })

  it("filters out entries without an id", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        json: async () => ({
          data: [
            { id: "job-1", updated_at: "x" },
            { updated_at: "y" }, // no id — must be filtered
            { id: "job-2", updated_at: "z" },
          ],
        }),
      }),
    )
    const result = await fetchSitemapJobs()
    expect(result.map((j) => j.id)).toEqual(["job-1", "job-2"])
  })
})
