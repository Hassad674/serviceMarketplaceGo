import { describe, it, expect, vi, beforeEach } from "vitest"
import {
  listModerationItems,
  approveMedia,
  rejectMedia,
  deleteMedia,
  approveMessageModeration,
  hideMessage,
  approveReviewModeration,
  deleteReview,
  resolveReport,
  restoreMessageModeration,
  restoreReviewModeration,
  restoreModerationGeneric,
} from "../api/moderation-api"

const mockAdminApi = vi.fn()

vi.mock("@/shared/lib/api-client", () => ({
  adminApi: (...a: unknown[]) => mockAdminApi(...a),
}))

beforeEach(() => {
  vi.clearAllMocks()
  mockAdminApi.mockResolvedValue({})
})

describe("admin moderation-api / list", () => {
  it("calls /admin/moderation with limit=20", () => {
    listModerationItems({ source: "", type: "", status: "", sort: "", page: 0 })
    expect(mockAdminApi).toHaveBeenCalledWith("/api/v1/admin/moderation?limit=20")
  })

  it("appends every filter present", () => {
    listModerationItems({
      source: "auto",
      type: "message",
      status: "pending",
      sort: "newest",
      page: 1,
    })
    const call = mockAdminApi.mock.calls[0][0] as string
    expect(call).toContain("source=auto")
    expect(call).toContain("type=message")
    expect(call).toContain("status=pending")
    expect(call).toContain("sort=newest")
    expect(call).toContain("page=1")
  })
})

describe("admin moderation-api / media actions", () => {
  it("approveMedia POSTs", () => {
    approveMedia("m-1")
    expect(mockAdminApi).toHaveBeenCalledWith(
      "/api/v1/admin/media/m-1/approve",
      { method: "POST", body: {} },
    )
  })

  it("rejectMedia POSTs", () => {
    rejectMedia("m-1")
    expect(mockAdminApi).toHaveBeenCalledWith(
      "/api/v1/admin/media/m-1/reject",
      { method: "POST", body: {} },
    )
  })

  it("deleteMedia DELETEs", () => {
    deleteMedia("m-1")
    expect(mockAdminApi).toHaveBeenCalledWith(
      "/api/v1/admin/media/m-1",
      { method: "DELETE" },
    )
  })
})

describe("admin moderation-api / message actions", () => {
  it("approveMessageModeration POSTs", () => {
    approveMessageModeration("msg-1")
    expect(mockAdminApi).toHaveBeenCalledWith(
      "/api/v1/admin/messages/msg-1/approve-moderation",
      { method: "POST", body: {} },
    )
  })

  it("hideMessage POSTs", () => {
    hideMessage("msg-1")
    expect(mockAdminApi).toHaveBeenCalledWith(
      "/api/v1/admin/messages/msg-1/hide",
      { method: "POST", body: {} },
    )
  })

  it("restoreMessageModeration POSTs", () => {
    restoreMessageModeration("msg-1")
    expect(mockAdminApi).toHaveBeenCalledWith(
      "/api/v1/admin/messages/msg-1/restore-moderation",
      { method: "POST", body: {} },
    )
  })
})

describe("admin moderation-api / review actions", () => {
  it("approveReviewModeration POSTs", () => {
    approveReviewModeration("rev-1")
    expect(mockAdminApi).toHaveBeenCalledWith(
      "/api/v1/admin/reviews/rev-1/approve-moderation",
      { method: "POST", body: {} },
    )
  })

  it("deleteReview DELETEs", () => {
    deleteReview("rev-1")
    expect(mockAdminApi).toHaveBeenCalledWith(
      "/api/v1/admin/reviews/rev-1",
      { method: "DELETE" },
    )
  })

  it("restoreReviewModeration POSTs", () => {
    restoreReviewModeration("rev-1")
    expect(mockAdminApi).toHaveBeenCalledWith(
      "/api/v1/admin/reviews/rev-1/restore-moderation",
      { method: "POST", body: {} },
    )
  })
})

describe("admin moderation-api / reports + generic", () => {
  it("resolveReport POSTs the resolution", () => {
    resolveReport("r-1", { status: "resolved", admin_note: "fixed" })
    expect(mockAdminApi).toHaveBeenCalledWith(
      "/api/v1/admin/reports/r-1/resolve",
      {
        method: "POST",
        body: { status: "resolved", admin_note: "fixed" },
      },
    )
  })

  it("resolveReport supports dismissed", () => {
    resolveReport("r-1", { status: "dismissed", admin_note: "n/a" })
    const body = (mockAdminApi.mock.calls[0][1] as { body: { status: string } }).body
    expect(body.status).toBe("dismissed")
  })

  it("restoreModerationGeneric uses the content_type and id", () => {
    restoreModerationGeneric("profile_bio", "u-1")
    expect(mockAdminApi).toHaveBeenCalledWith(
      "/api/v1/admin/moderation/profile_bio/u-1/restore",
      { method: "POST", body: {} },
    )
  })
})
