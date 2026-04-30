import { describe, it, expect, vi, beforeEach } from "vitest"
import {
  listOpenJobs,
  applyToJob,
  withdrawApplication,
  listJobApplications,
  listMyApplications,
  contactApplicant,
  hasApplied,
} from "../job-application-api"

const mockApiClient = vi.fn()

vi.mock("@/shared/lib/api-client", () => ({
  apiClient: (...a: unknown[]) => mockApiClient(...a),
}))

beforeEach(() => {
  vi.clearAllMocks()
  mockApiClient.mockResolvedValue({})
})

describe("job-application-api / listOpenJobs", () => {
  it("calls /open with no query when filters/cursor empty", () => {
    listOpenJobs()
    expect(mockApiClient).toHaveBeenCalledWith("/api/v1/jobs/open")
  })

  it("appends search filter", () => {
    listOpenJobs({ search: "react dev" })
    const call = mockApiClient.mock.calls[0][0] as string
    expect(call).toContain("search=react+dev")
  })

  it("appends applicant_type and budget_type filters", () => {
    listOpenJobs({
      applicant_type: "freelance",
      budget_type: "fixed",
    })
    const call = mockApiClient.mock.calls[0][0] as string
    expect(call).toContain("applicant_type=freelance")
    expect(call).toContain("budget_type=fixed")
  })

  it("appends min/max budget filters", () => {
    listOpenJobs({ min_budget: 1000, max_budget: 5000 })
    const call = mockApiClient.mock.calls[0][0] as string
    expect(call).toContain("min_budget=1000")
    expect(call).toContain("max_budget=5000")
  })

  it("joins skills with commas", () => {
    listOpenJobs({ skills: ["go", "react", "k8s"] })
    const call = mockApiClient.mock.calls[0][0] as string
    expect(call).toContain("skills=go%2Creact%2Ck8s")
  })

  it("appends cursor", () => {
    listOpenJobs(undefined, "tok1")
    const call = mockApiClient.mock.calls[0][0] as string
    expect(call).toContain("cursor=tok1")
  })

  it("does not append min_budget=0 as undefined", () => {
    listOpenJobs({ min_budget: 0 })
    const call = mockApiClient.mock.calls[0][0] as string
    expect(call).toContain("min_budget=0")
  })
})

describe("job-application-api / applications", () => {
  it("applyToJob POSTs the message + optional video_url", () => {
    applyToJob("j-1", { message: "hi", video_url: "https://v" })
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/jobs/j-1/apply",
      {
        method: "POST",
        body: { message: "hi", video_url: "https://v" },
      },
    )
  })

  it("withdrawApplication DELETEs by application id", () => {
    withdrawApplication("a-1")
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/jobs/applications/a-1",
      { method: "DELETE" },
    )
  })

  it("listJobApplications GETs scoped to a job", () => {
    listJobApplications("j-1")
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/jobs/j-1/applications",
    )
  })

  it("listJobApplications appends cursor", () => {
    listJobApplications("j-1", "tok")
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/jobs/j-1/applications?cursor=tok",
    )
  })

  it("listMyApplications without cursor", () => {
    listMyApplications()
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/jobs/applications/mine",
    )
  })

  it("contactApplicant POSTs the contact endpoint", () => {
    contactApplicant("j-1", "u-2")
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/jobs/j-1/applications/u-2/contact",
      { method: "POST" },
    )
  })

  it("hasApplied GETs /has-applied", () => {
    hasApplied("j-1")
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/jobs/j-1/has-applied",
    )
  })
})

describe("job-application-api / errors", () => {
  it("propagates errors from listOpenJobs", async () => {
    mockApiClient.mockRejectedValueOnce(new Error("502"))
    await expect(listOpenJobs()).rejects.toThrow("502")
  })

  it("propagates errors from applyToJob", async () => {
    mockApiClient.mockRejectedValueOnce(new Error("403"))
    await expect(applyToJob("j-1", { message: "" })).rejects.toThrow("403")
  })
})
