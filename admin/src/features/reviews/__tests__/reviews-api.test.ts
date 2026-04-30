import { describe, it, expect, vi, beforeEach } from "vitest"
import {
  listAdminReviews,
  getAdminReview,
  deleteAdminReview,
} from "../api/reviews-api"

const mockAdminApi = vi.fn()

vi.mock("@/shared/lib/api-client", () => ({
  adminApi: (...a: unknown[]) => mockAdminApi(...a),
}))

beforeEach(() => {
  vi.clearAllMocks()
  mockAdminApi.mockResolvedValue({})
})

describe("admin reviews-api", () => {
  it("listAdminReviews calls /admin/reviews with limit=20", () => {
    listAdminReviews({ search: "", rating: "", sort: "", filter: "", page: 0 })
    expect(mockAdminApi).toHaveBeenCalledWith("/api/v1/admin/reviews?limit=20")
  })

  it("listAdminReviews appends search/rating/sort/filter/page", () => {
    listAdminReviews({
      search: "great",
      rating: "5",
      sort: "newest",
      filter: "flagged",
      page: 2,
    })
    const call = mockAdminApi.mock.calls[0][0] as string
    expect(call).toContain("search=great")
    expect(call).toContain("rating=5")
    expect(call).toContain("sort=newest")
    expect(call).toContain("filter=flagged")
    expect(call).toContain("page=2")
  })

  it("getAdminReview GETs by id", () => {
    getAdminReview("rev-1")
    expect(mockAdminApi).toHaveBeenCalledWith("/api/v1/admin/reviews/rev-1")
  })

  it("deleteAdminReview DELETEs by id", () => {
    deleteAdminReview("rev-1")
    expect(mockAdminApi).toHaveBeenCalledWith(
      "/api/v1/admin/reviews/rev-1",
      { method: "DELETE" },
    )
  })

  it("propagates errors", async () => {
    mockAdminApi.mockRejectedValueOnce(new Error("403"))
    await expect(getAdminReview("rev-x")).rejects.toThrow("403")
  })
})
