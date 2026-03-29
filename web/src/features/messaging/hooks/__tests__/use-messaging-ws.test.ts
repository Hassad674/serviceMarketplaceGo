import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import { renderHook, act } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { createElement } from "react"
import { useMessagingWS } from "../use-messaging-ws"

// --- Mock WebSocket ---

type MockWSInstance = {
  onopen: (() => void) | null
  onmessage: ((event: { data: string }) => void) | null
  onclose: (() => void) | null
  onerror: (() => void) | null
  readyState: number
  send: ReturnType<typeof vi.fn>
  close: ReturnType<typeof vi.fn>
}

let mockWSInstance: MockWSInstance

class MockWebSocket {
  static OPEN = 1
  static CLOSED = 3
  static CONNECTING = 0

  onopen: (() => void) | null = null
  onmessage: ((event: { data: string }) => void) | null = null
  onclose: (() => void) | null = null
  onerror: (() => void) | null = null
  readyState = MockWebSocket.CONNECTING
  send = vi.fn()
  close = vi.fn(() => {
    ;(this as unknown as MockWSInstance).readyState = MockWebSocket.CLOSED
  })

  constructor() {
    mockWSInstance = this as unknown as MockWSInstance
  }
}

// Mock modules before tests
vi.mock("@/shared/hooks/use-unread-count", () => ({
  unreadCountQueryKey: (uid: string | undefined) => ["user", uid, "messaging", "unread-count"],
  UNREAD_COUNT_QUERY_KEY: ["messaging", "unread-count"],
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

// Helper: flush the async getWSUrl() microtask so the WebSocket is created.
// getWSUrl is async (returns Promise), even in dev where it resolves immediately.
async function flushWSConnect() {
  await act(async () => {
    await Promise.resolve()
    await Promise.resolve()
  })
}

describe("useMessagingWS", () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.stubGlobal("WebSocket", MockWebSocket)
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.restoreAllMocks()
  })

  it("does not connect when userId is undefined", async () => {
    renderHook(() => useMessagingWS(undefined), {
      wrapper: createWrapper(),
    })

    expect(mockWSInstance).toBeUndefined()
  })

  it("connects and sets isConnected to true on open", async () => {
    const { result } = renderHook(() => useMessagingWS("user-1"), {
      wrapper: createWrapper(),
    })
    await flushWSConnect()

    // Flush the async getWSUrl() microtask so WebSocket is created
    await act(async () => {
      await vi.runAllTimersAsync()
    })

    expect(result.current.isConnected).toBe(false)

    // Simulate connection open
    act(() => {
      mockWSInstance.readyState = MockWebSocket.OPEN
      mockWSInstance.onopen?.()
    })

    expect(result.current.isConnected).toBe(true)
  })

  it("sends heartbeat on interval after connection opens", async () => {
    renderHook(() => useMessagingWS("user-1"), {
      wrapper: createWrapper(),
    })
    await flushWSConnect()

    act(() => {
      mockWSInstance.readyState = MockWebSocket.OPEN
      mockWSInstance.onopen?.()
    })

    // Advance past heartbeat interval (30s)
    act(() => {
      vi.advanceTimersByTime(30_000)
    })

    expect(mockWSInstance.send).toHaveBeenCalledWith(
      JSON.stringify({ type: "heartbeat" }),
    )
  })

  it("handles new_message frame and updates typing state", async () => {
    const { result } = renderHook(() => useMessagingWS("user-1"), {
      wrapper: createWrapper(),
    })
    await flushWSConnect()

    act(() => {
      mockWSInstance.readyState = MockWebSocket.OPEN
      mockWSInstance.onopen?.()
    })

    // First, set a typing indicator
    act(() => {
      mockWSInstance.onmessage?.({
        data: JSON.stringify({
          type: "typing",
          payload: { conversation_id: "conv-1", user_id: "user-2" },
        }),
      })
    })

    expect(result.current.typingUsers["conv-1"]).toBeDefined()

    // new_message should clear typing for that conversation
    act(() => {
      mockWSInstance.onmessage?.({
        data: JSON.stringify({
          type: "new_message",
          payload: {
            id: "msg-1",
            conversation_id: "conv-1",
            sender_id: "user-2",
            content: "Hello",
            type: "text",
            metadata: null,
            seq: 1,
            status: "sent",
            edited_at: null,
            deleted_at: null,
            created_at: "2026-03-26T10:00:00Z",
          },
        }),
      })
    })

    expect(result.current.typingUsers["conv-1"]).toBeUndefined()
  })

  it("handles typing frame from another user", async () => {
    const { result } = renderHook(() => useMessagingWS("user-1"), {
      wrapper: createWrapper(),
    })
    await flushWSConnect()

    act(() => {
      mockWSInstance.readyState = MockWebSocket.OPEN
      mockWSInstance.onopen?.()
    })

    act(() => {
      mockWSInstance.onmessage?.({
        data: JSON.stringify({
          type: "typing",
          payload: { conversation_id: "conv-1", user_id: "user-2" },
        }),
      })
    })

    expect(result.current.typingUsers["conv-1"]).toEqual({
      userId: "user-2",
    })
  })

  it("ignores typing frame from self", async () => {
    const { result } = renderHook(() => useMessagingWS("user-1"), {
      wrapper: createWrapper(),
    })
    await flushWSConnect()

    act(() => {
      mockWSInstance.readyState = MockWebSocket.OPEN
      mockWSInstance.onopen?.()
    })

    act(() => {
      mockWSInstance.onmessage?.({
        data: JSON.stringify({
          type: "typing",
          payload: { conversation_id: "conv-1", user_id: "user-1" },
        }),
      })
    })

    expect(result.current.typingUsers["conv-1"]).toBeUndefined()
  })

  it("clears typing after delay", async () => {
    const { result } = renderHook(() => useMessagingWS("user-1"), {
      wrapper: createWrapper(),
    })
    await flushWSConnect()

    act(() => {
      mockWSInstance.readyState = MockWebSocket.OPEN
      mockWSInstance.onopen?.()
    })

    act(() => {
      mockWSInstance.onmessage?.({
        data: JSON.stringify({
          type: "typing",
          payload: { conversation_id: "conv-1", user_id: "user-2" },
        }),
      })
    })

    expect(result.current.typingUsers["conv-1"]).toBeDefined()

    // Advance past typing clear delay (5s)
    act(() => {
      vi.advanceTimersByTime(5_000)
    })

    expect(result.current.typingUsers["conv-1"]).toBeUndefined()
  })

  it("handles unread_count frame", async () => {
    const { result } = renderHook(() => useMessagingWS("user-1"), {
      wrapper: createWrapper(),
    })
    await flushWSConnect()

    act(() => {
      mockWSInstance.readyState = MockWebSocket.OPEN
      mockWSInstance.onopen?.()
    })

    act(() => {
      mockWSInstance.onmessage?.({
        data: JSON.stringify({
          type: "unread_count",
          payload: { count: 7 },
        }),
      })
    })

    expect(result.current.totalUnread).toBe(7)
  })

  it("handles presence frame without crashing", async () => {
    const { result } = renderHook(() => useMessagingWS("user-1"), {
      wrapper: createWrapper(),
    })
    await flushWSConnect()

    act(() => {
      mockWSInstance.readyState = MockWebSocket.OPEN
      mockWSInstance.onopen?.()
    })

    // presence frame should not throw
    act(() => {
      mockWSInstance.onmessage?.({
        data: JSON.stringify({
          type: "presence",
          payload: { user_id: "user-2", online: true },
        }),
      })
    })

    // No crash means success; presence updates query cache
    expect(result.current.isConnected).toBe(true)
  })

  it("handles status_update frame without crashing", async () => {
    const { result } = renderHook(() => useMessagingWS("user-1"), {
      wrapper: createWrapper(),
    })
    await flushWSConnect()

    act(() => {
      mockWSInstance.readyState = MockWebSocket.OPEN
      mockWSInstance.onopen?.()
    })

    act(() => {
      mockWSInstance.onmessage?.({
        data: JSON.stringify({
          type: "status_update",
          payload: {
            conversation_id: "conv-1",
            reader_id: "user-2",
            up_to_seq: 5,
            status: "read",
          },
        }),
      })
    })

    expect(result.current.isConnected).toBe(true)
  })

  it("ignores malformed frames", async () => {
    const { result } = renderHook(() => useMessagingWS("user-1"), {
      wrapper: createWrapper(),
    })
    await flushWSConnect()

    act(() => {
      mockWSInstance.readyState = MockWebSocket.OPEN
      mockWSInstance.onopen?.()
    })

    // Should not throw on invalid JSON
    act(() => {
      mockWSInstance.onmessage?.({ data: "not json" })
    })

    expect(result.current.isConnected).toBe(true)
  })

  it("sets isConnected to false on close", async () => {
    const { result } = renderHook(() => useMessagingWS("user-1"), {
      wrapper: createWrapper(),
    })
    await flushWSConnect()

    act(() => {
      mockWSInstance.readyState = MockWebSocket.OPEN
      mockWSInstance.onopen?.()
    })

    expect(result.current.isConnected).toBe(true)

    act(() => {
      mockWSInstance.readyState = MockWebSocket.CLOSED
      mockWSInstance.onclose?.()
    })

    expect(result.current.isConnected).toBe(false)
  })

  it("schedules reconnect on close with increasing delay", async () => {
    const { result } = renderHook(() => useMessagingWS("user-1"), {
      wrapper: createWrapper(),
    })
    await flushWSConnect()

    act(() => {
      mockWSInstance.readyState = MockWebSocket.OPEN
      mockWSInstance.onopen?.()
    })

    expect(result.current.isConnected).toBe(true)

    const closedInstance = mockWSInstance

    // Close the connection (set readyState to CLOSED first, like real WS)
    act(() => {
      mockWSInstance.readyState = MockWebSocket.CLOSED
      mockWSInstance.onclose?.()
    })

    // isConnected should be false immediately
    expect(result.current.isConnected).toBe(false)

    // Advance timers to trigger reconnect (delay = 1000ms * 2^0 = 1000ms)
    await act(async () => {
      vi.advanceTimersByTime(1500)
    })
    await flushWSConnect()

    // After reconnect, a new WS instance was created (different object)
    expect(mockWSInstance).not.toBe(closedInstance)
  })

  it("sends sync on reconnect when lastSeqMap has entries", async () => {
    const { result } = renderHook(() => useMessagingWS("user-1"), {
      wrapper: createWrapper(),
    })
    await flushWSConnect()

    act(() => {
      mockWSInstance.readyState = MockWebSocket.OPEN
      mockWSInstance.onopen?.()
    })

    // Receive a message to populate lastSeqMap
    act(() => {
      mockWSInstance.onmessage?.({
        data: JSON.stringify({
          type: "new_message",
          payload: {
            id: "msg-1",
            conversation_id: "conv-1",
            sender_id: "user-2",
            content: "Hello",
            type: "text",
            metadata: null,
            seq: 5,
            status: "sent",
            edited_at: null,
            deleted_at: null,
            created_at: "2026-03-26T10:00:00Z",
          },
        }),
      })
    })

    const firstInstance = mockWSInstance

    // Close and reconnect (set readyState to CLOSED first, like real WS)
    act(() => {
      firstInstance.readyState = MockWebSocket.CLOSED
      firstInstance.onclose?.()
    })

    act(() => {
      vi.advanceTimersByTime(1500)
    })

    // Simulate the new connection opening
    act(() => {
      mockWSInstance.readyState = MockWebSocket.OPEN
      mockWSInstance.onopen?.()
    })

    // Should have sent a sync frame with the lastSeqMap
    const syncCalls = mockWSInstance.send.mock.calls.filter(
      (call: unknown[]) => {
        const parsed = JSON.parse(call[0] as string)
        return parsed.type === "sync"
      },
    )

    expect(syncCalls.length).toBe(1)
    const syncFrame = JSON.parse(syncCalls[0][0] as string)
    expect(syncFrame.conversations["conv-1"]).toBe(5)
  })

  it("sendTyping sends typing frame", async () => {
    const { result } = renderHook(() => useMessagingWS("user-1"), {
      wrapper: createWrapper(),
    })
    await flushWSConnect()

    act(() => {
      mockWSInstance.readyState = MockWebSocket.OPEN
      mockWSInstance.onopen?.()
    })

    act(() => {
      result.current.sendTyping("conv-1")
    })

    expect(mockWSInstance.send).toHaveBeenCalledWith(
      JSON.stringify({ type: "typing", conversation_id: "conv-1" }),
    )
  })

  it("cleans up on unmount", async () => {
    const { unmount } = renderHook(() => useMessagingWS("user-1"), {
      wrapper: createWrapper(),
    })
    await flushWSConnect()

    act(() => {
      mockWSInstance.readyState = MockWebSocket.OPEN
      mockWSInstance.onopen?.()
    })

    const instance = mockWSInstance

    unmount()

    expect(instance.close).toHaveBeenCalled()
  })

  it("handles message_edited frame without crashing", async () => {
    const { result } = renderHook(() => useMessagingWS("user-1"), {
      wrapper: createWrapper(),
    })
    await flushWSConnect()

    act(() => {
      mockWSInstance.readyState = MockWebSocket.OPEN
      mockWSInstance.onopen?.()
    })

    act(() => {
      mockWSInstance.onmessage?.({
        data: JSON.stringify({
          type: "message_edited",
          payload: {
            id: "msg-1",
            conversation_id: "conv-1",
            sender_id: "user-2",
            content: "Edited content",
            type: "text",
            metadata: null,
            seq: 1,
            status: "sent",
            edited_at: "2026-03-26T10:05:00Z",
            deleted_at: null,
            created_at: "2026-03-26T10:00:00Z",
          },
        }),
      })
    })

    expect(result.current.isConnected).toBe(true)
  })

  it("handles message_deleted frame without crashing", async () => {
    const { result } = renderHook(() => useMessagingWS("user-1"), {
      wrapper: createWrapper(),
    })
    await flushWSConnect()

    act(() => {
      mockWSInstance.readyState = MockWebSocket.OPEN
      mockWSInstance.onopen?.()
    })

    act(() => {
      mockWSInstance.onmessage?.({
        data: JSON.stringify({
          type: "message_deleted",
          payload: { message_id: "msg-1", conversation_id: "conv-1" },
        }),
      })
    })

    expect(result.current.isConnected).toBe(true)
  })
})
