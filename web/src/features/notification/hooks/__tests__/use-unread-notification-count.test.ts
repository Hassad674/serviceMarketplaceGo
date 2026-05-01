import { describe, it, expect, vi, beforeEach } from "vitest"
import { renderHook, waitFor } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { createElement } from "react"
import {
  useUnreadNotificationCount,
  unreadNotifCountKey,
} from "../use-unread-notification-count"

const mockGetUnreadNotificationCount = vi.fn()

vi.mock("../../api/notification-api", () => ({
  getUnreadNotificationCount: (...args: unknown[]) =>
    mockGetUnreadNotificationCount(...args),
}))

vi.mock("@/shared/hooks/use-current-user-id", () => ({
  useCurrentUserId: () => "test-uid",
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

describe("unreadNotifCountKey", () => {
  it("builds key with user id", () => {
    expect(unreadNotifCountKey("u-1")).toEqual([
      "user",
      "u-1",
      "notifications",
      "unread-count",
    ])
  })

  it("builds key with undefined user id", () => {
    expect(unreadNotifCountKey(undefined)).toEqual([
      "user",
      undefined,
      "notifications",
      "unread-count",
    ])
  })
})

describe("useUnreadNotificationCount", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("returns unread count via select", async () => {
    mockGetUnreadNotificationCount.mockResolvedValue({
      data: { count: 7 },
    })

    const { result } = renderHook(() => useUnreadNotificationCount(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data).toBe(7)
  })

  it("returns zero count", async () => {
    mockGetUnreadNotificationCount.mockResolvedValue({
      data: { count: 0 },
    })

    const { result } = renderHook(() => useUnreadNotificationCount(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data).toBe(0)
  })

  it("handles error", async () => {
    mockGetUnreadNotificationCount.mockRejectedValue(new Error("fail"))

    const { result } = renderHook(() => useUnreadNotificationCount(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isError).toBe(true))
  })
})
