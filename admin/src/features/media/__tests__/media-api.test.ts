import { describe, it, expect, vi, beforeEach } from "vitest"
import {
  listAdminMedia,
  getAdminMedia,
  approveMedia,
  rejectMedia,
  deleteMedia,
} from "../api/media-api"

const mockAdminApi = vi.fn()

vi.mock("@/shared/lib/api-client", () => ({
  adminApi: (...a: unknown[]) => mockAdminApi(...a),
}))

beforeEach(() => {
  vi.clearAllMocks()
  mockAdminApi.mockResolvedValue({})
})

describe("admin media-api / listAdminMedia", () => {
  it("calls /admin/media with limit=20 by default", () => {
    listAdminMedia({ status: "", type: "", context: "", search: "", sort: "", page: 0 })
    expect(mockAdminApi).toHaveBeenCalledWith("/api/v1/admin/media?limit=20")
  })

  it("appends every filter present", () => {
    listAdminMedia({
      status: "pending",
      type: "image",
      context: "review",
      search: "kw",
      sort: "newest",
      page: 5,
    })
    const call = mockAdminApi.mock.calls[0][0] as string
    expect(call).toContain("status=pending")
    expect(call).toContain("type=image")
    expect(call).toContain("context=review")
    expect(call).toContain("search=kw")
    expect(call).toContain("sort=newest")
    expect(call).toContain("page=5")
  })
})

describe("admin media-api / single media", () => {
  it("getAdminMedia GETs by id", () => {
    getAdminMedia("m-1")
    expect(mockAdminApi).toHaveBeenCalledWith("/api/v1/admin/media/m-1")
  })

  it("approveMedia POSTs the approve endpoint", () => {
    approveMedia("m-1")
    expect(mockAdminApi).toHaveBeenCalledWith(
      "/api/v1/admin/media/m-1/approve",
      { method: "POST", body: {} },
    )
  })

  it("rejectMedia POSTs the reject endpoint", () => {
    rejectMedia("m-1")
    expect(mockAdminApi).toHaveBeenCalledWith(
      "/api/v1/admin/media/m-1/reject",
      { method: "POST", body: {} },
    )
  })

  it("deleteMedia DELETEs by id", () => {
    deleteMedia("m-1")
    expect(mockAdminApi).toHaveBeenCalledWith(
      "/api/v1/admin/media/m-1",
      { method: "DELETE" },
    )
  })
})

describe("admin media-api / errors", () => {
  it("propagates errors", async () => {
    mockAdminApi.mockRejectedValueOnce(new Error("404"))
    await expect(getAdminMedia("missing")).rejects.toThrow("404")
  })
})
