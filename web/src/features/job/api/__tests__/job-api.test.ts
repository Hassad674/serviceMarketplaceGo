import { describe, it, expect, vi, beforeEach } from "vitest"
import {
  createJob,
  updateJob,
  getJob,
  listMyJobs,
  closeJob,
  reopenJob,
  deleteJob,
  markApplicationsViewed,
  getCredits,
} from "../job-api"

const mockApiClient = vi.fn()

vi.mock("@/shared/lib/api-client", () => ({
  apiClient: (...a: unknown[]) => mockApiClient(...a),
}))

const baseJob = {
  title: "Need a dev",
  description: "...",
  skills: ["go"],
  applicant_type: "freelance",
  budget_type: "fixed",
  min_budget: 1000,
  max_budget: 5000,
  is_indefinite: false,
  description_type: "text",
}

beforeEach(() => {
  vi.clearAllMocks()
  mockApiClient.mockResolvedValue({})
})

describe("job-api", () => {
  it("createJob POSTs to /jobs", async () => {
    await createJob(baseJob)
    expect(mockApiClient).toHaveBeenCalledWith("/api/v1/jobs", {
      method: "POST",
      body: baseJob,
    })
  })

  it("updateJob PUTs to /jobs/:id", async () => {
    await updateJob("j-1", baseJob)
    expect(mockApiClient).toHaveBeenCalledWith("/api/v1/jobs/j-1", {
      method: "PUT",
      body: baseJob,
    })
  })

  it("getJob GETs by id", () => {
    getJob("j-1")
    expect(mockApiClient).toHaveBeenCalledWith("/api/v1/jobs/j-1")
  })

  it("listMyJobs without cursor", () => {
    listMyJobs()
    expect(mockApiClient).toHaveBeenCalledWith("/api/v1/jobs/mine")
  })

  it("listMyJobs with cursor (URL-encoded)", () => {
    listMyJobs("a/b")
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/jobs/mine?cursor=a%2Fb",
    )
  })

  it("closeJob POSTs /close", () => {
    closeJob("j-1")
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/jobs/j-1/close",
      { method: "POST" },
    )
  })

  it("reopenJob POSTs /reopen", () => {
    reopenJob("j-1")
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/jobs/j-1/reopen",
      { method: "POST" },
    )
  })

  it("deleteJob DELETEs /jobs/:id", () => {
    deleteJob("j-1")
    expect(mockApiClient).toHaveBeenCalledWith("/api/v1/jobs/j-1", {
      method: "DELETE",
    })
  })

  it("markApplicationsViewed POSTs /mark-viewed", () => {
    markApplicationsViewed("j-1")
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/jobs/j-1/mark-viewed",
      { method: "POST" },
    )
  })

  it("getCredits GETs /credits", () => {
    getCredits()
    expect(mockApiClient).toHaveBeenCalledWith("/api/v1/jobs/credits")
  })

  it("propagates createJob errors", async () => {
    mockApiClient.mockRejectedValueOnce(new Error("422"))
    await expect(createJob(baseJob)).rejects.toThrow("422")
  })

  it("propagates getJob errors", async () => {
    mockApiClient.mockRejectedValueOnce(new Error("404"))
    await expect(getJob("missing")).rejects.toThrow("404")
  })
})
