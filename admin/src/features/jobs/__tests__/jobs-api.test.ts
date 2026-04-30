import { describe, it, expect, vi, beforeEach } from "vitest"
import {
  listAdminJobs,
  getAdminJob,
  deleteAdminJob,
  listAdminJobApplications,
  deleteAdminJobApplication,
} from "../api/jobs-api"

const mockAdminApi = vi.fn()

vi.mock("@/shared/lib/api-client", () => ({
  adminApi: (...a: unknown[]) => mockAdminApi(...a),
}))

beforeEach(() => {
  vi.clearAllMocks()
  mockAdminApi.mockResolvedValue({})
})

describe("admin jobs-api / listAdminJobs", () => {
  it("calls /admin/jobs with limit=20", () => {
    listAdminJobs({ status: "", search: "", sort: "", filter: "", page: 0 })
    expect(mockAdminApi).toHaveBeenCalledWith("/api/v1/admin/jobs?limit=20")
  })

  it("appends status, search, sort, filter, page", () => {
    listAdminJobs({
      status: "open",
      search: "react",
      sort: "newest",
      filter: "fraud",
      page: 2,
    })
    const call = mockAdminApi.mock.calls[0][0] as string
    expect(call).toContain("status=open")
    expect(call).toContain("search=react")
    expect(call).toContain("sort=newest")
    expect(call).toContain("filter=fraud")
    expect(call).toContain("page=2")
    expect(call).toContain("limit=20")
  })

  it("getAdminJob GETs by id", () => {
    getAdminJob("j-1")
    expect(mockAdminApi).toHaveBeenCalledWith("/api/v1/admin/jobs/j-1")
  })

  it("deleteAdminJob DELETEs by id", () => {
    deleteAdminJob("j-1")
    expect(mockAdminApi).toHaveBeenCalledWith(
      "/api/v1/admin/jobs/j-1",
      { method: "DELETE" },
    )
  })
})

describe("admin jobs-api / job-applications", () => {
  it("listAdminJobApplications calls /admin/job-applications with limit=20", () => {
    listAdminJobApplications({ job_id: "", search: "", sort: "", filter: "", page: 0 })
    expect(mockAdminApi).toHaveBeenCalledWith(
      "/api/v1/admin/job-applications?limit=20",
    )
  })

  it("appends job_id when present", () => {
    listAdminJobApplications({ job_id: "j-1", search: "", sort: "", filter: "", page: 0 })
    const call = mockAdminApi.mock.calls[0][0] as string
    expect(call).toContain("job_id=j-1")
  })

  it("deleteAdminJobApplication DELETEs by id", () => {
    deleteAdminJobApplication("a-1")
    expect(mockAdminApi).toHaveBeenCalledWith(
      "/api/v1/admin/job-applications/a-1",
      { method: "DELETE" },
    )
  })
})

describe("admin jobs-api / errors", () => {
  it("propagates errors", async () => {
    mockAdminApi.mockRejectedValueOnce(new Error("500"))
    await expect(getAdminJob("j-1")).rejects.toThrow("500")
  })
})
