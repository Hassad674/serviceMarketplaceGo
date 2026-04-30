import { describe, it, expect, vi, beforeEach } from "vitest"
import {
  fetchReviewsByUser,
  fetchAverageRating,
  fetchCanReview,
  createReview,
  uploadReviewVideo,
} from "../review-api"

const mockApiClient = vi.fn()

vi.mock("@/shared/lib/api-client", () => ({
  apiClient: (...a: unknown[]) => mockApiClient(...a),
  API_BASE_URL: "",
}))

beforeEach(() => {
  vi.clearAllMocks()
  mockApiClient.mockResolvedValue({})
})

describe("review-api / fetchReviewsByUser", () => {
  it("calls without cursor", async () => {
    await fetchReviewsByUser("u-1")
    expect(mockApiClient).toHaveBeenCalledWith("/api/v1/reviews/user/u-1")
  })

  it("appends cursor when provided", async () => {
    await fetchReviewsByUser("u-1", "abc")
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/reviews/user/u-1?cursor=abc",
    )
  })
})

describe("review-api / fetchAverageRating", () => {
  it("calls /average/:id", () => {
    fetchAverageRating("u-1")
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/reviews/average/u-1",
    )
  })
})

describe("review-api / fetchCanReview", () => {
  it("calls /can-review/:id", () => {
    fetchCanReview("p-1")
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/reviews/can-review/p-1",
    )
  })
})

describe("review-api / createReview", () => {
  it("POSTs the payload to /reviews", () => {
    createReview({
      proposal_id: "p-1",
      global_rating: 5,
      comment: "great",
    })
    expect(mockApiClient).toHaveBeenCalledWith("/api/v1/reviews", {
      method: "POST",
      body: expect.objectContaining({
        proposal_id: "p-1",
        global_rating: 5,
      }),
    })
  })

  it("supports optional sub-rating fields", () => {
    createReview({
      proposal_id: "p-1",
      global_rating: 4,
      timeliness: 5,
      communication: 4,
      quality: 4,
    })
    const body = (mockApiClient.mock.calls[0][1] as { body: { timeliness: number } }).body
    expect(body.timeliness).toBe(5)
  })

  it("supports title_visible toggle", () => {
    createReview({
      proposal_id: "p-1",
      global_rating: 3,
      title_visible: true,
    })
    const body = (mockApiClient.mock.calls[0][1] as { body: { title_visible: boolean } }).body
    expect(body.title_visible).toBe(true)
  })
})

describe("review-api / uploadReviewVideo", () => {
  it("uploads via POST to /api/v1/upload/review-video and returns the url", async () => {
    const mockFetch = vi.fn(async () => ({
      ok: true,
      json: async () => ({ url: "https://cdn/v.mp4" }),
    }))
    vi.stubGlobal("fetch", mockFetch)

    const file = new File(["x"], "v.mp4", { type: "video/mp4" })
    const url = await uploadReviewVideo(file)
    expect(url).toBe("https://cdn/v.mp4")
    expect(mockFetch).toHaveBeenCalled()

    vi.unstubAllGlobals()
  })

  it("throws when upload fails (non-2xx)", async () => {
    const mockFetch = vi.fn(async () => ({
      ok: false,
      json: async () => ({ message: "too big" }),
    }))
    vi.stubGlobal("fetch", mockFetch)

    const file = new File(["x"], "v.mp4", { type: "video/mp4" })
    await expect(uploadReviewVideo(file)).rejects.toThrow("too big")
    vi.unstubAllGlobals()
  })

  it("uses generic error when JSON parse fails", async () => {
    const mockFetch = vi.fn(async () => ({
      ok: false,
      json: async () => {
        throw new Error("not json")
      },
    }))
    vi.stubGlobal("fetch", mockFetch)
    const file = new File(["x"], "v.mp4", { type: "video/mp4" })
    await expect(uploadReviewVideo(file)).rejects.toThrow("Upload failed")
    vi.unstubAllGlobals()
  })
})
