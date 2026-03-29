import { describe, it, expect, vi, beforeEach } from "vitest"
import { renderHook, waitFor } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { createElement } from "react"
import { useNotifications, notificationsQueryKey } from "../use-notifications"

const mockListNotifications = vi.fn()

vi.mock("../../api/notification-api", () => ({
  listNotifications: (...args: unknown[]) => mockListNotifications(...args),
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
  return ({ children }: { children: React.ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children)
}

describe("useNotifications", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("calls listNotifications API on mount", async () => {
    mockListNotifications.mockResolvedValue({
      data: [],
      next_cursor: "",
      has_more: false,
    })

    const { result } = renderHook(() => useNotifications(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(mockListNotifications).toHaveBeenCalledOnce()
  })

  it("returns notification data from API", async () => {
    mockListNotifications.mockResolvedValue({
      data: [
        {
          id: "notif-1",
          user_id: "test-user-id",
          type: "proposal_received",
          title: "New proposal",
          body: "You received a new proposal",
          data: { proposal_id: "prop-1" },
          read_at: null,
          created_at: "2026-03-28T09:00:00Z",
        },
        {
          id: "notif-2",
          user_id: "test-user-id",
          type: "new_message",
          title: "New message",
          body: "You have a new message",
          data: { conversation_id: "conv-1" },
          read_at: "2026-03-28T10:00:00Z",
          created_at: "2026-03-28T08:00:00Z",
        },
      ],
      next_cursor: "cursor-xyz",
      has_more: true,
    })

    const { result } = renderHook(() => useNotifications(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    const page = result.current.data?.pages[0]
    expect(page?.data).toHaveLength(2)
    expect(page?.data[0].title).toBe("New proposal")
    expect(page?.has_more).toBe(true)
  })

  it("supports fetching next page", async () => {
    mockListNotifications
      .mockResolvedValueOnce({
        data: [{ id: "notif-1", type: "proposal_received", title: "First", body: "", data: {}, read_at: null, created_at: "2026-03-28T09:00:00Z", user_id: "test-user-id" }],
        next_cursor: "cursor-page2",
        has_more: true,
      })
      .mockResolvedValueOnce({
        data: [{ id: "notif-2", type: "new_message", title: "Second", body: "", data: {}, read_at: null, created_at: "2026-03-28T08:00:00Z", user_id: "test-user-id" }],
        next_cursor: "",
        has_more: false,
      })

    const { result } = renderHook(() => useNotifications(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.hasNextPage).toBe(true)

    result.current.fetchNextPage()
    await waitFor(() => expect(result.current.data?.pages).toHaveLength(2))
    expect(mockListNotifications).toHaveBeenCalledTimes(2)
    expect(mockListNotifications).toHaveBeenLastCalledWith("cursor-page2")
  })

  it("builds user-scoped query key", () => {
    expect(notificationsQueryKey("uid-456")).toEqual([
      "user",
      "uid-456",
      "notifications",
    ])
  })
})
