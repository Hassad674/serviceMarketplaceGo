import { describe, it, expect, vi, beforeEach } from "vitest"
import { renderHook, waitFor, act } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { createElement } from "react"
import { useMessages, useSendMessage, useEditMessage, useDeleteMessage, MESSAGES_KEY_BASE, messagesQueryKey } from "../use-messages"

// Mock the API
const mockListMessages = vi.fn()
const mockSendMessage = vi.fn()
const mockEditMessage = vi.fn()
const mockDeleteMessage = vi.fn()

vi.mock("../../api/messaging-api", () => ({
  listMessages: (...args: unknown[]) => mockListMessages(...args),
  sendMessage: (...args: unknown[]) => mockSendMessage(...args),
  editMessage: (...args: unknown[]) => mockEditMessage(...args),
  deleteMessage: (...args: unknown[]) => mockDeleteMessage(...args),
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
  return ({ children }: { children: React.ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children)
}

describe("useMessages", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("does not fetch when conversationId is null", () => {
    renderHook(() => useMessages(null), {
      wrapper: createWrapper(),
    })

    expect(mockListMessages).not.toHaveBeenCalled()
  })

  it("fetches messages when conversationId provided", async () => {
    mockListMessages.mockResolvedValue({
      data: [{ id: "msg-1", content: "Hello" }],
      has_more: false,
    })

    const { result } = renderHook(() => useMessages("conv-1"), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    expect(mockListMessages).toHaveBeenCalledWith("conv-1", undefined)
  })

  it("uses correct query key format", () => {
    expect(MESSAGES_KEY_BASE).toBe("messaging-messages")
    expect(messagesQueryKey("uid-1", "conv-1")).toEqual(["user", "uid-1", "messaging-messages", "conv-1"])
  })
})

describe("useSendMessage", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("calls sendMessage API with correct params", async () => {
    const newMsg = {
      id: "msg-new",
      conversation_id: "conv-1",
      sender_id: "user-1",
      content: "Hello!",
      type: "text",
      metadata: null,
      seq: 10,
      status: "sent",
      edited_at: null,
      deleted_at: null,
      created_at: new Date().toISOString(),
    }
    mockSendMessage.mockResolvedValue(newMsg)

    const { result } = renderHook(() => useSendMessage("conv-1"), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      result.current.mutate({ content: "Hello!" })
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    expect(mockSendMessage).toHaveBeenCalledWith("conv-1", "Hello!", undefined, undefined, undefined)
  })
})

describe("useEditMessage", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("calls editMessage API with messageId and content", async () => {
    const updatedMsg = {
      id: "msg-1",
      content: "Updated content",
      edited_at: new Date().toISOString(),
    }
    mockEditMessage.mockResolvedValue(updatedMsg)

    const { result } = renderHook(() => useEditMessage("conv-1"), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      result.current.mutate({ messageId: "msg-1", content: "Updated content" })
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    expect(mockEditMessage).toHaveBeenCalledWith("msg-1", "Updated content")
  })
})

describe("useDeleteMessage", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("calls deleteMessage API with messageId", async () => {
    mockDeleteMessage.mockResolvedValue(undefined)

    const { result } = renderHook(() => useDeleteMessage("conv-1"), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      result.current.mutate("msg-1")
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    expect(mockDeleteMessage).toHaveBeenCalledWith("msg-1")
  })
})
