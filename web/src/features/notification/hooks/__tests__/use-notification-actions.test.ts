import { describe, it, expect, vi, beforeEach } from "vitest"
import { renderHook, waitFor, act } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { createElement } from "react"
import { useMarkAsRead, useMarkAllAsRead, useDeleteNotification } from "../use-notification-actions"

const mockMarkNotificationAsRead = vi.fn()
const mockMarkAllNotificationsAsRead = vi.fn()
const mockDeleteNotification = vi.fn()

vi.mock("../../api/notification-api", () => ({
  markNotificationAsRead: (...args: unknown[]) => mockMarkNotificationAsRead(...args),
  markAllNotificationsAsRead: (...args: unknown[]) => mockMarkAllNotificationsAsRead(...args),
  deleteNotification: (...args: unknown[]) => mockDeleteNotification(...args),
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
  const Wrapper = ({ children }: { children: React.ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children)
  Wrapper.displayName = "TestWrapper"
  return Wrapper
}

describe("useMarkAsRead", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("calls markNotificationAsRead API with notification id", async () => {
    mockMarkNotificationAsRead.mockResolvedValue(undefined)

    const { result } = renderHook(() => useMarkAsRead(), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      result.current.mutate("notif-1")
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(mockMarkNotificationAsRead).toHaveBeenCalledWith("notif-1")
  })

  it("handles API error gracefully", async () => {
    mockMarkNotificationAsRead.mockRejectedValue(new Error("Network error"))

    const { result } = renderHook(() => useMarkAsRead(), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      result.current.mutate("notif-bad")
    })

    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(result.current.error?.message).toBe("Network error")
  })
})

describe("useMarkAllAsRead", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("calls markAllNotificationsAsRead API", async () => {
    mockMarkAllNotificationsAsRead.mockResolvedValue(undefined)

    const { result } = renderHook(() => useMarkAllAsRead(), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      result.current.mutate()
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(mockMarkAllNotificationsAsRead).toHaveBeenCalledOnce()
  })
})

describe("useDeleteNotification", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("calls deleteNotification API with notification id", async () => {
    mockDeleteNotification.mockResolvedValue(undefined)

    const { result } = renderHook(() => useDeleteNotification(), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      result.current.mutate("notif-del-1")
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(mockDeleteNotification).toHaveBeenCalledWith("notif-del-1")
  })
})
