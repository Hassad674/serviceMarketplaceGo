import { describe, it, expect, vi, beforeEach } from "vitest"
import { renderHook, waitFor } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { createElement } from "react"
import { useReviewsByUser, useAverageRating } from "../use-reviews"

const mockFetchReviewsByUser = vi.fn()
const mockFetchAverageRating = vi.fn()

vi.mock("../../api/review-api", () => ({
  fetchReviewsByUser: (...args: unknown[]) => mockFetchReviewsByUser(...args),
  fetchAverageRating: (...args: unknown[]) => mockFetchAverageRating(...args),
  fetchCanReview: vi.fn(),
  createReview: vi.fn(),
  uploadReviewVideo: vi.fn(),
}))

vi.mock("@/shared/hooks/use-current-user-id", () => ({
  useCurrentUserId: () => "test-user-id",
}))

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0 },
    },
  })
  const Wrapper = ({ children }: { children: React.ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children)
  Wrapper.displayName = "TestWrapper"
  return Wrapper
}

describe("useReviewsByUser", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("calls fetchReviewsByUser API with userId", async () => {
    mockFetchReviewsByUser.mockResolvedValue({
      data: [],
      next_cursor: "",
      has_more: false,
    })

    const { result } = renderHook(() => useReviewsByUser("user-42"), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(mockFetchReviewsByUser).toHaveBeenCalledWith("user-42")
  })

  it("returns review data from API", async () => {
    mockFetchReviewsByUser.mockResolvedValue({
      data: [
        {
          id: "rev-1",
          proposal_id: "prop-1",
          reviewer_id: "reviewer-1",
          reviewed_id: "user-42",
          global_rating: 4,
          timeliness: 5,
          communication: 4,
          quality: 4,
          comment: "Great work!",
          video_url: null,
          created_at: "2026-03-20T12:00:00Z",
        },
        {
          id: "rev-2",
          proposal_id: "prop-2",
          reviewer_id: "reviewer-2",
          reviewed_id: "user-42",
          global_rating: 5,
          timeliness: null,
          communication: null,
          quality: null,
          comment: "Excellent",
          video_url: null,
          created_at: "2026-03-22T14:00:00Z",
        },
      ],
      next_cursor: "",
      has_more: false,
    })

    const { result } = renderHook(() => useReviewsByUser("user-42"), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data?.data).toHaveLength(2)
    expect(result.current.data?.data[0].comment).toBe("Great work!")
    expect(result.current.data?.data[1].global_rating).toBe(5)
  })
})

describe("useAverageRating", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("calls fetchAverageRating API with userId", async () => {
    mockFetchAverageRating.mockResolvedValue({
      data: { average: 4.5, count: 12 },
    })

    const { result } = renderHook(() => useAverageRating("user-42"), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(mockFetchAverageRating).toHaveBeenCalledWith("user-42")
  })

  it("returns average rating and count", async () => {
    mockFetchAverageRating.mockResolvedValue({
      data: { average: 4.2, count: 8 },
    })

    const { result } = renderHook(() => useAverageRating("user-42"), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data?.data.average).toBe(4.2)
    expect(result.current.data?.data.count).toBe(8)
  })

  it("returns zero values for user with no reviews", async () => {
    mockFetchAverageRating.mockResolvedValue({
      data: { average: 0, count: 0 },
    })

    const { result } = renderHook(() => useAverageRating("user-new"), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data?.data.average).toBe(0)
    expect(result.current.data?.data.count).toBe(0)
  })
})
