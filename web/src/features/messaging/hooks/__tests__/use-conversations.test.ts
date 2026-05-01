import { describe, it, expect, vi, beforeEach } from "vitest"
import { renderHook, waitFor } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { createElement } from "react"
import { useConversations, conversationsQueryKey } from "../use-conversations"

// Mock the API
const mockListConversations = vi.fn()

vi.mock("../../api/messaging-api", () => ({
  listConversations: (...args: unknown[]) => mockListConversations(...args),
}))

vi.mock("@/shared/hooks/use-current-user-id", () => ({
  useCurrentUserId: () => "test-user-id",
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
  const Wrapper = ({ children }: { children: React.ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children)
  Wrapper.displayName = "TestWrapper"
  return Wrapper
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
          other_org_id: "org-2",
          other_org_name: "Alice",
          other_org_type: "provider_personal",
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
    expect(result.current.data?.data[0].other_org_name).toBe("Alice")
  })

  it("builds user-scoped query key", () => {
    expect(conversationsQueryKey("uid-123")).toEqual(["user", "uid-123", "messaging", "conversations"])
  })
})
