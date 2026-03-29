import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import { renderHook, act } from "@testing-library/react"
import { createElement } from "react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"

// Mock the notification imports to avoid cross-feature dependency resolution
vi.mock("@/features/notification/hooks/use-unread-notification-count", () => ({
  unreadNotifCountKey: (uid: string | undefined) => ["user", uid, "notifications", "unread-count"],
}))

vi.mock("@/features/notification/hooks/use-notifications", () => ({
  notificationsQueryKey: (uid: string | undefined) => ["user", uid, "notifications"],
}))

vi.mock("sonner", () => ({
  toast: vi.fn(),
}))

import { toast } from "sonner"

type WSHandler = ((event: { data: string }) => void) | null

class MockWebSocket {
  static CONNECTING = 0
  static OPEN = 1
  static CLOSING = 2
  static CLOSED = 3

  url: string
  readyState = MockWebSocket.CONNECTING
  onopen: (() => void) | null = null
  onmessage: WSHandler = null
  onclose: (() => void) | null = null
  onerror: (() => void) | null = null

  send = vi.fn()
  close = vi.fn(() => {
    this.readyState = MockWebSocket.CLOSED
    if (this.onclose) this.onclose()
  })

  constructor(url: string) {
    this.url = url
    // Store instance for test access
    MockWebSocket.lastInstance = this
  }

  // Simulate opening the connection
  simulateOpen() {
    this.readyState = MockWebSocket.OPEN
    if (this.onopen) this.onopen()
  }

  // Simulate receiving a message
  simulateMessage(data: Record<string, unknown>) {
    if (this.onmessage) {
      this.onmessage({ data: JSON.stringify(data) })
    }
  }

  // Simulate an error
  simulateError() {
    if (this.onerror) this.onerror()
  }

  static lastInstance: MockWebSocket | null = null
}

// Attach static constants to the constructor for readyState comparisons
Object.assign(MockWebSocket, {
  CONNECTING: 0,
  OPEN: 1,
  CLOSING: 2,
  CLOSED: 3,
})

let queryClient: QueryClient

function createWrapper() {
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return createElement(QueryClientProvider, { client: queryClient }, children)
  }
}

beforeEach(() => {
  vi.useFakeTimers()
  MockWebSocket.lastInstance = null
  vi.stubGlobal("WebSocket", MockWebSocket)
  vi.stubGlobal("fetch", vi.fn(() => Promise.resolve({ ok: false })))

  queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
})

afterEach(() => {
  vi.useRealTimers()
  vi.restoreAllMocks()
})

describe("useGlobalWS", () => {
  // Import after mocks are set up
  async function importHook() {
    const mod = await import("../use-global-ws")
    return mod.useGlobalWS
  }

  it("does not connect when userId is undefined", async () => {
    const useGlobalWS = await importHook()

    renderHook(() => useGlobalWS(undefined), { wrapper: createWrapper() })

    expect(MockWebSocket.lastInstance).toBeNull()
  })

  it("connects when userId is provided", async () => {
    const useGlobalWS = await importHook()

    renderHook(() => useGlobalWS("user-123"), { wrapper: createWrapper() })

    // Allow the async connect to resolve
    await act(async () => {
      await vi.runAllTimersAsync()
    })

    expect(MockWebSocket.lastInstance).not.toBeNull()
    expect(MockWebSocket.lastInstance!.url).toContain("/api/v1/ws")
  })

  it("sends heartbeat after connection opens", async () => {
    const useGlobalWS = await importHook()

    renderHook(() => useGlobalWS("user-123"), { wrapper: createWrapper() })

    await act(async () => {
      await vi.runAllTimersAsync()
    })

    const ws = MockWebSocket.lastInstance!
    act(() => {
      ws.simulateOpen()
    })

    // Advance past heartbeat interval (30 seconds)
    act(() => {
      vi.advanceTimersByTime(30_000)
    })

    expect(ws.send).toHaveBeenCalledWith(JSON.stringify({ type: "heartbeat" }))
  })

  it("updates unread count on unread_count message", async () => {
    const useGlobalWS = await importHook()

    renderHook(() => useGlobalWS("user-123"), { wrapper: createWrapper() })

    await act(async () => {
      await vi.runAllTimersAsync()
    })

    const ws = MockWebSocket.lastInstance!
    act(() => {
      ws.simulateOpen()
    })

    const setDataSpy = vi.spyOn(queryClient, "setQueryData")

    act(() => {
      ws.simulateMessage({ type: "unread_count", payload: { count: 7 } })
    })

    expect(setDataSpy).toHaveBeenCalledWith(
      ["user", "user-123", "messaging", "unread-count"],
      { count: 7 },
    )
  })

  it("shows toast and updates cache on notification message", async () => {
    const useGlobalWS = await importHook()

    renderHook(() => useGlobalWS("user-123"), { wrapper: createWrapper() })

    await act(async () => {
      await vi.runAllTimersAsync()
    })

    const ws = MockWebSocket.lastInstance!
    act(() => {
      ws.simulateOpen()
    })

    act(() => {
      ws.simulateMessage({
        type: "notification",
        payload: { title: "New message", body: "You got mail" },
      })
    })

    expect(toast).toHaveBeenCalledWith("New message", {
      description: "You got mail",
    })
  })

  it("invokes call event handler on call_event message", async () => {
    const useGlobalWS = await importHook()
    const callHandler = vi.fn()

    renderHook(() => useGlobalWS("user-123", callHandler), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      await vi.runAllTimersAsync()
    })

    const ws = MockWebSocket.lastInstance!
    act(() => {
      ws.simulateOpen()
    })

    const callPayload = { callId: "call-1", action: "ringing" }
    act(() => {
      ws.simulateMessage({ type: "call_event", payload: callPayload })
    })

    expect(callHandler).toHaveBeenCalledWith(callPayload)
  })

  it("returns setMessagingPageActive function", async () => {
    const useGlobalWS = await importHook()

    const { result } = renderHook(() => useGlobalWS("user-123"), {
      wrapper: createWrapper(),
    })

    expect(result.current.setMessagingPageActive).toBeTypeOf("function")
  })

  it("cleans up WebSocket on unmount", async () => {
    const useGlobalWS = await importHook()

    const { unmount } = renderHook(() => useGlobalWS("user-123"), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      await vi.runAllTimersAsync()
    })

    const ws = MockWebSocket.lastInstance!
    act(() => {
      ws.simulateOpen()
    })

    unmount()

    expect(ws.close).toHaveBeenCalled()
  })

  it("ignores malformed messages without crashing", async () => {
    const useGlobalWS = await importHook()

    renderHook(() => useGlobalWS("user-123"), { wrapper: createWrapper() })

    await act(async () => {
      await vi.runAllTimersAsync()
    })

    const ws = MockWebSocket.lastInstance!
    act(() => {
      ws.simulateOpen()
    })

    // Send invalid JSON — should not throw
    expect(() => {
      act(() => {
        if (ws.onmessage) {
          ws.onmessage({ data: "not json at all" })
        }
      })
    }).not.toThrow()
  })
})
