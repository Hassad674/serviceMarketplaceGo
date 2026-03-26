import { describe, it, expect, vi, beforeEach } from "vitest"
import { renderHook, waitFor } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { createElement } from "react"
import { useConversations, CONVERSATIONS_QUERY_KEY } from "../use-conversations"

// Mock the API
const mockListConversations = vi.fn()

vi.mock("../../api/messaging-api", () => ({
  listConversations: (...args: unknown[]) => mockListConversations(...args),
}))

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
        gcTime: 0,
      },
    },
  })
  return ({ children }: { children: React.ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children)
}

describe("useConversations", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("calls listConversations API on mount", async () => {
    mockListConversations.mockResolvedValue({
      data: [],
      has_more: false,
    })

    const { result } = renderHook(() => useConversations(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    expect(mockListConversations).toHaveBeenCalledOnce()
  })

  it("returns conversation data from API", async () => {
    mockListConversations.mockResolvedValue({
      data: [
        {
          id: "conv-1",
          other_user_id: "user-2",
          other_user_name: "Alice",
          other_user_role: "provider",
          other_photo_url: "",
          last_message: "Hello",
          last_message_at: "2026-03-26T10:00:00Z",
          unread_count: 2,
          last_message_seq: 5,
          online: true,
        },
      ],
      has_more: false,
    })

    const { result } = renderHook(() => useConversations(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    expect(result.current.data?.data).toHaveLength(1)
    expect(result.current.data?.data[0].other_user_name).toBe("Alice")
  })

  it("uses correct query key", () => {
    expect(CONVERSATIONS_QUERY_KEY).toEqual(["messaging", "conversations"])
  })
})
