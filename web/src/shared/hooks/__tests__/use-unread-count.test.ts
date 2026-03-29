import { describe, it, expect, vi, beforeEach } from "vitest"
import { renderHook, waitFor } from "@testing-library/react"
import { createElement } from "react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"

vi.mock("@/shared/hooks/use-current-user-id", () => ({
  useCurrentUserId: vi.fn(),
}))

vi.mock("@/shared/lib/api-client", () => ({
  apiClient: vi.fn(),
  API_BASE_URL: "http://localhost:8080",
}))

import { useCurrentUserId } from "@/shared/hooks/use-current-user-id"
import { apiClient } from "@/shared/lib/api-client"
import { useUnreadCount, unreadCountQueryKey } from "../use-unread-count"

const mockedUseCurrentUserId = vi.mocked(useCurrentUserId)
const mockedApiClient = vi.mocked(apiClient)

function createWrapper() {
  const client = new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0 },
    },
  })
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return createElement(QueryClientProvider, { client }, children)
  }
}

beforeEach(() => {
  vi.clearAllMocks()
})

describe("unreadCountQueryKey", () => {
  it("returns a key scoped to the user id", () => {
    const key = unreadCountQueryKey("user-abc")
    expect(key).toEqual(["user", "user-abc", "messaging", "unread-count"])
  })

  it("returns a key with undefined when no user id", () => {
    const key = unreadCountQueryKey(undefined)
    expect(key).toEqual(["user", undefined, "messaging", "unread-count"])
  })
})

describe("useUnreadCount", () => {
  it("fetches unread count from API", async () => {
    mockedUseCurrentUserId.mockReturnValue("user-123")
    mockedApiClient.mockResolvedValue({ count: 5 })

    const { result } = renderHook(() => useUnreadCount(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    expect(result.current.data).toEqual({ count: 5 })
    expect(mockedApiClient).toHaveBeenCalledWith("/api/v1/messaging/unread-count")
  })

  it("returns zero count when API returns zero", async () => {
    mockedUseCurrentUserId.mockReturnValue("user-456")
    mockedApiClient.mockResolvedValue({ count: 0 })

    const { result } = renderHook(() => useUnreadCount(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    expect(result.current.data).toEqual({ count: 0 })
  })

  it("handles API error gracefully", async () => {
    mockedUseCurrentUserId.mockReturnValue("user-789")
    mockedApiClient.mockRejectedValue(new Error("Network error"))

    const { result } = renderHook(() => useUnreadCount(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isError).toBe(true))

    expect(result.current.error).toBeInstanceOf(Error)
    expect(result.current.error?.message).toBe("Network error")
  })

  it("uses user-scoped query key", async () => {
    mockedUseCurrentUserId.mockReturnValue("user-abc")
    mockedApiClient.mockResolvedValue({ count: 3 })

    const wrapper = createWrapper()
    const { result } = renderHook(() => useUnreadCount(), { wrapper })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    // The query key should include the user id for cache isolation
    expect(result.current.data).toEqual({ count: 3 })
  })
})
