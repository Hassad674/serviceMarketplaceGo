import { describe, it, expect, vi, afterEach } from "vitest"

vi.mock("@/shared/lib/api-client", () => ({
  API_BASE_URL: "http://localhost:8080",
}))

import { fetchJobForMetadata } from "../job-server"

afterEach(() => {
  vi.restoreAllMocks()
})

describe("fetchJobForMetadata — PERF-W-06 JobPosting JSON-LD source", () => {
  it("hits /api/v1/jobs/:id with ISR", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({
        id: "job-1",
        creator_id: "user-1",
        title: "Build API",
        description: "Backend Go work",
        skills: ["go"],
        applicant_type: "all",
        budget_type: "one_shot",
        min_budget: 1000,
        max_budget: 5000,
        status: "open",
        created_at: "2026-04-01T00:00:00Z",
        updated_at: "2026-04-01T00:00:00Z",
        is_indefinite: false,
        description_type: "text",
      }),
    })
    vi.stubGlobal("fetch", fetchMock)

    const result = await fetchJobForMetadata("job-1")
    expect(fetchMock.mock.calls[0][0]).toBe(
      "http://localhost:8080/api/v1/jobs/job-1",
    )
    const init = fetchMock.mock.calls[0][1] as { next?: { revalidate?: number } }
    expect(init.next?.revalidate).toBe(120)
    expect(result?.title).toBe("Build API")
  })

  it("returns null on non-200", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({ ok: false, json: async () => ({}) }),
    )
    const result = await fetchJobForMetadata("job-x")
    expect(result).toBeNull()
  })

  it("returns null on network error", async () => {
    vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new Error("net")))
    const result = await fetchJobForMetadata("job-x")
    expect(result).toBeNull()
  })
})
