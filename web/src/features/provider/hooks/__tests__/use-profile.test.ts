import { describe, it, expect, vi, beforeEach } from "vitest"
import { renderHook, waitFor, act } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { createElement } from "react"
import { useProfile, useUpdateProfile } from "../use-profile"

const mockGetMyProfile = vi.fn()
const mockUpdateProfile = vi.fn()

vi.mock("../../api/profile-api", () => ({
  getMyProfile: (...args: unknown[]) => mockGetMyProfile(...args),
  updateProfile: (...args: unknown[]) => mockUpdateProfile(...args),
}))

vi.mock("@/shared/hooks/use-current-user-id", () => ({
  useCurrentUserId: () => "test-user-id",
}))

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0 },
      mutations: { retry: false },
    },
  })
  return ({ children }: { children: React.ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children)
}

describe("useProfile", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("calls getMyProfile API on mount", async () => {
    mockGetMyProfile.mockResolvedValue({
      user_id: "test-user-id",
      title: "Full-Stack Developer",
      photo_url: "",
      presentation_video_url: "",
      referrer_video_url: "",
      about: "",
      referrer_about: "",
      created_at: "2026-03-20T10:00:00Z",
      updated_at: "2026-03-20T10:00:00Z",
    })

    const { result } = renderHook(() => useProfile(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(mockGetMyProfile).toHaveBeenCalledOnce()
  })

  it("returns profile data from API", async () => {
    mockGetMyProfile.mockResolvedValue({
      user_id: "test-user-id",
      title: "Senior Backend Engineer",
      photo_url: "https://storage.example.com/photos/avatar.jpg",
      presentation_video_url: "https://storage.example.com/videos/intro.mp4",
      referrer_video_url: "",
      about: "Experienced Go developer with 10 years of expertise.",
      referrer_about: "",
      created_at: "2026-03-20T10:00:00Z",
      updated_at: "2026-03-25T14:00:00Z",
    })

    const { result } = renderHook(() => useProfile(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data?.title).toBe("Senior Backend Engineer")
    expect(result.current.data?.about).toBe(
      "Experienced Go developer with 10 years of expertise.",
    )
    expect(result.current.data?.photo_url).toBe(
      "https://storage.example.com/photos/avatar.jpg",
    )
  })

  it("returns profile with empty optional fields", async () => {
    mockGetMyProfile.mockResolvedValue({
      user_id: "test-user-id",
      title: "",
      photo_url: "",
      presentation_video_url: "",
      referrer_video_url: "",
      about: "",
      referrer_about: "",
      created_at: "2026-03-20T10:00:00Z",
      updated_at: "2026-03-20T10:00:00Z",
    })

    const { result } = renderHook(() => useProfile(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data?.title).toBe("")
    expect(result.current.data?.about).toBe("")
  })
})

describe("useUpdateProfile", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("calls updateProfile API on mutate", async () => {
    mockUpdateProfile.mockResolvedValue({
      user_id: "test-user-id",
      title: "Lead Developer",
      photo_url: "",
      presentation_video_url: "",
      referrer_video_url: "",
      about: "Updated bio.",
      referrer_about: "",
      created_at: "2026-03-20T10:00:00Z",
      updated_at: "2026-03-28T09:00:00Z",
    })

    const { result } = renderHook(() => useUpdateProfile(), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      result.current.mutate({ title: "Lead Developer", about: "Updated bio." })
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(mockUpdateProfile).toHaveBeenCalledWith({
      title: "Lead Developer",
      about: "Updated bio.",
    })
  })

  it("returns updated profile data", async () => {
    mockUpdateProfile.mockResolvedValue({
      user_id: "test-user-id",
      title: "CTO",
      photo_url: "",
      presentation_video_url: "",
      referrer_video_url: "",
      about: "Now leading tech.",
      referrer_about: "",
      created_at: "2026-03-20T10:00:00Z",
      updated_at: "2026-03-28T09:00:00Z",
    })

    const { result } = renderHook(() => useUpdateProfile(), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      result.current.mutate({ title: "CTO", about: "Now leading tech." })
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data?.title).toBe("CTO")
  })

  it("handles update failure", async () => {
    mockUpdateProfile.mockRejectedValue(new Error("Validation failed"))

    const { result } = renderHook(() => useUpdateProfile(), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      result.current.mutate({ title: "" })
    })

    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(result.current.error?.message).toBe("Validation failed")
  })
})
