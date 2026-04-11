import { describe, it, expect, vi, beforeEach } from "vitest"
import { renderHook, waitFor, act } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { createElement } from "react"
import { useStartConversation } from "../use-start-conversation"

// Mock the API
const mockStartConversation = vi.fn()

vi.mock("../../api/messaging-api", () => ({
  startConversation: (...args: unknown[]) => mockStartConversation(...args),
}))

// Mock the router
const mockPush = vi.fn()

vi.mock("@i18n/navigation", () => ({
  useRouter: () => ({
    push: mockPush,
  }),
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

describe("useStartConversation", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("calls startConversation API with correct params", async () => {
    mockStartConversation.mockResolvedValue({
      conversation_id: "conv-new",
      message: { id: "msg-1", content: "Hello!" },
    })

    const { result } = renderHook(() => useStartConversation(), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      result.current.mutate({ otherOrgId: "org-2", content: "Hello!" })
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    expect(mockStartConversation).toHaveBeenCalledWith("org-2", "Hello!")
  })

  it("navigates to conversation on success", async () => {
    mockStartConversation.mockResolvedValue({
      conversation_id: "conv-new",
      message: { id: "msg-1", content: "Hello!" },
    })

    const { result } = renderHook(() => useStartConversation(), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      result.current.mutate({ otherOrgId: "org-2", content: "Hi!" })
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    expect(mockPush).toHaveBeenCalledWith("/messages?id=conv-new")
  })

  it("does not navigate on error", async () => {
    mockStartConversation.mockRejectedValue(new Error("API error"))

    const { result } = renderHook(() => useStartConversation(), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      result.current.mutate({ otherOrgId: "org-2", content: "Hello!" })
    })

    await waitFor(() => expect(result.current.isError).toBe(true))

    expect(mockPush).not.toHaveBeenCalled()
  })
})
