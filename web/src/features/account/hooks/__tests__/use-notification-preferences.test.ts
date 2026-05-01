import { describe, it, expect, vi, beforeEach } from "vitest"
import { renderHook, waitFor, act } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { createElement } from "react"
import {
  useNotificationPreferences,
  useUpdateNotificationPreferences,
} from "../use-notification-preferences"

const mockApiClient = vi.fn()

vi.mock("@/shared/lib/api-client", () => ({
  apiClient: (...args: unknown[]) => mockApiClient(...args),
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

describe("useNotificationPreferences", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("fetches preferences on mount", async () => {
    const prefs = [
      { type: "new_message", in_app: true, push: true, email: false },
      { type: "proposal", in_app: true, push: false, email: true },
    ]
    // Hook returns the full envelope (consumers read both `data` and
    // `email_notifications_enabled` — see notification-settings.tsx).
    mockApiClient.mockResolvedValue({ data: prefs, email_notifications_enabled: true })

    const { result } = renderHook(() => useNotificationPreferences(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data?.data).toEqual(prefs)
    expect(result.current.data?.email_notifications_enabled).toBe(true)
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/notifications/preferences",
    )
  })

  it("returns empty array when no preferences", async () => {
    mockApiClient.mockResolvedValue({ data: [], email_notifications_enabled: false })

    const { result } = renderHook(() => useNotificationPreferences(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data?.data).toEqual([])
  })

  it("handles fetch error", async () => {
    mockApiClient.mockRejectedValue(new Error("Forbidden"))

    const { result } = renderHook(() => useNotificationPreferences(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isError).toBe(true))
  })
})

describe("useUpdateNotificationPreferences", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("sends preferences update", async () => {
    mockApiClient.mockResolvedValue(undefined)

    const { result } = renderHook(() => useUpdateNotificationPreferences(), {
      wrapper: createWrapper(),
    })

    const prefs = [
      { type: "new_message", in_app: true, push: false, email: false },
    ]

    await act(async () => {
      result.current.mutate(prefs)
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/notifications/preferences",
      { method: "PUT", body: { preferences: prefs } },
    )
  })

  it("handles update failure", async () => {
    mockApiClient.mockRejectedValue(new Error("Server error"))

    const { result } = renderHook(() => useUpdateNotificationPreferences(), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      result.current.mutate([])
    })

    await waitFor(() => expect(result.current.isError).toBe(true))
  })
})
