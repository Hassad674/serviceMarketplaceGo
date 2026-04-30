/**
 * CallSlot tests — PERF-W-01.
 *
 * Contract under test:
 *   1. CallSlot renders nothing visible while idle (no LiveKit chunk
 *      should be loaded).
 *   2. The first WS `call_incoming` event arms the slot, mounts the
 *      runtime, and forwards the event to it.
 *   3. The first `startCall(...)` invocation through the exposed
 *      CallContext arms the slot and forwards the args to the runtime.
 *   4. Repeated registrations of the WS handler swap correctly so the
 *      slot's pre-mount handler is replaced by the runtime's once it
 *      mounts.
 *
 * The `./call-runtime` module is mocked so we never touch the real
 * `livekit-client` module — the whole point of the lazy boundary.
 */

import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import { render, screen, waitFor, act } from "@testing-library/react"
import { useContext, useEffect } from "react"
import { CallContext } from "@/shared/hooks/use-call-context"
import type { CallEventPayload } from "../../types"

// Spy that captures the props passed into the lazy CallRuntime so the
// test can assert what the slot forwards.
const runtimeRenderSpy = vi.fn()

vi.mock("../call-runtime", () => ({
  CallRuntime: (props: Record<string, unknown>) => {
    runtimeRenderSpy(props)
    // Mimic the runtime's contract: replace the WS handler with its
    // own (here a no-op) so the swap path is exercised. Also expose
    // an `onIdle` trigger so the test can assert the slot resets its
    // pending-event state when the runtime signals "I consumed it".
    const register = props.registerCallEventHandler as (
      h: ((p: CallEventPayload) => void) | undefined,
    ) => void
    const onIdle = props.onIdle as () => void
    useEffect(() => {
      register(() => {})
      return () => register(undefined)
    }, [register])
    return (
      <div data-testid="call-runtime">
        runtime mounted
        <button type="button" data-testid="idle-trigger" onClick={onIdle}>
          idle
        </button>
      </div>
    )
  },
}))

import { CallSlot } from "../call-slot"

beforeEach(() => {
  runtimeRenderSpy.mockReset()
})

afterEach(() => {
  vi.restoreAllMocks()
})

describe("CallSlot — PERF-W-01 lazy boundary", () => {
  it("renders nothing visible while idle (no runtime mounted)", () => {
    const register = vi.fn()
    render(<CallSlot registerCallEventHandler={register} />)

    expect(screen.queryByTestId("call-runtime")).toBeNull()
    expect(runtimeRenderSpy).not.toHaveBeenCalled()
  })

  it("renders provided children even while idle", () => {
    const register = vi.fn()
    render(
      <CallSlot registerCallEventHandler={register}>
        <span data-testid="child">hello</span>
      </CallSlot>,
    )

    expect(screen.getByTestId("child")).toBeInTheDocument()
    expect(screen.queryByTestId("call-runtime")).toBeNull()
  })

  it("registers a pre-mount handler with the global WS bridge", () => {
    const register = vi.fn()
    render(<CallSlot registerCallEventHandler={register} />)

    // Slot should have called register with a function on mount.
    expect(register).toHaveBeenCalled()
    // First call must be a function — represents the slot's
    // pre-mount handler.
    const firstCallArg = register.mock.calls[0]?.[0]
    expect(typeof firstCallArg).toBe("function")
  })

  it("ignores non-incoming WS events (idle stays idle)", () => {
    let captured: ((p: CallEventPayload) => void) | undefined
    const register = vi.fn((handler) => {
      if (typeof handler === "function") captured = handler
    })

    render(<CallSlot registerCallEventHandler={register} />)
    expect(captured).toBeDefined()

    // call_accepted must NOT trigger the runtime — only call_incoming
    // is allowed to wake the lazy boundary (the outgoing path uses
    // startCall via context).
    act(() => {
      captured!({
        event: "call_accepted",
        call_id: "x",
        conversation_id: "c",
        initiator_id: "i",
        recipient_id: "r",
        call_type: "audio",
      })
    })

    expect(runtimeRenderSpy).not.toHaveBeenCalled()
  })

  it("arms the runtime on first call_incoming event", async () => {
    let captured: ((p: CallEventPayload) => void) | undefined
    const register = vi.fn((handler) => {
      if (typeof handler === "function") captured = handler
    })

    render(<CallSlot registerCallEventHandler={register} />)

    const incoming: CallEventPayload = {
      event: "call_incoming",
      call_id: "abc",
      conversation_id: "conv-1",
      initiator_id: "user-2",
      recipient_id: "user-1",
      call_type: "video",
      initiator_name: "Alice",
    }

    act(() => {
      captured!(incoming)
    })

    await waitFor(() => {
      expect(screen.getByTestId("call-runtime")).toBeInTheDocument()
    })

    // The runtime received the captured incoming event for replay.
    const props = runtimeRenderSpy.mock.calls.at(-1)?.[0] as Record<string, unknown>
    expect(props.pendingIncomingEvent).toEqual(incoming)
    expect(props.pendingStartCall).toBeNull()
  })

  it("arms the runtime on first startCall via context", async () => {
    function StartCallButton() {
      const ctx = useContext(CallContext)
      return (
        <button
          type="button"
          onClick={() => {
            ctx?.startCall("conv-9", "user-9", "Bob", "audio")
          }}
        >
          go
        </button>
      )
    }

    const register = vi.fn()
    render(
      <CallSlot registerCallEventHandler={register}>
        <StartCallButton />
      </CallSlot>,
    )

    const btn = screen.getByText("go")
    await act(async () => {
      btn.click()
    })

    await waitFor(() => {
      expect(screen.getByTestId("call-runtime")).toBeInTheDocument()
    })

    const props = runtimeRenderSpy.mock.calls.at(-1)?.[0] as Record<string, unknown>
    expect(props.pendingStartCall).toEqual({
      conversationId: "conv-9",
      recipientId: "user-9",
      recipientName: "Bob",
      callType: "audio",
    })
    expect(props.pendingIncomingEvent).toBeNull()
  })

  it("startCall defaults missing args to '' / 'audio'", async () => {
    function StartCallButton() {
      const ctx = useContext(CallContext)
      return (
        <button
          type="button"
          onClick={() => {
            ctx?.startCall("conv-9", "user-9")
          }}
        >
          go
        </button>
      )
    }

    const register = vi.fn()
    render(
      <CallSlot registerCallEventHandler={register}>
        <StartCallButton />
      </CallSlot>,
    )

    const btn = screen.getByText("go")
    await act(async () => {
      btn.click()
    })

    await waitFor(() => {
      expect(screen.getByTestId("call-runtime")).toBeInTheDocument()
    })

    const props = runtimeRenderSpy.mock.calls.at(-1)?.[0] as Record<string, unknown>
    expect(props.pendingStartCall).toEqual({
      conversationId: "conv-9",
      recipientId: "user-9",
      recipientName: "",
      callType: "audio",
    })
  })

  it("CallContext is null until the slot is on the tree", () => {
    function ContextProbe({ onValue }: { onValue: (v: unknown) => void }) {
      const ctx = useContext(CallContext)
      onValue(ctx)
      return null
    }

    const probe = vi.fn()
    render(<ContextProbe onValue={probe} />)
    expect(probe).toHaveBeenLastCalledWith(null)
  })

  it("CallContext becomes available once children render under the slot", () => {
    function ContextProbe({ onValue }: { onValue: (v: unknown) => void }) {
      const ctx = useContext(CallContext)
      onValue(ctx)
      return null
    }

    const probe = vi.fn()
    const register = vi.fn()
    render(
      <CallSlot registerCallEventHandler={register}>
        <ContextProbe onValue={probe} />
      </CallSlot>,
    )

    const last = probe.mock.calls.at(-1)?.[0] as { startCall?: unknown } | null
    expect(last).toBeTruthy()
    expect(typeof last?.startCall).toBe("function")
  })

  it("handleIdle clears pending state once the runtime signals consumption", async () => {
    let captured: ((p: CallEventPayload) => void) | undefined
    const register = vi.fn((handler) => {
      if (typeof handler === "function") captured = handler
    })

    render(<CallSlot registerCallEventHandler={register} />)

    act(() => {
      captured!({
        event: "call_incoming",
        call_id: "abc",
        conversation_id: "conv-1",
        initiator_id: "user-2",
        recipient_id: "user-1",
        call_type: "audio",
      })
    })

    await waitFor(() => {
      expect(screen.getByTestId("call-runtime")).toBeInTheDocument()
    })

    // Trigger the runtime's onIdle hook — slot must clear its
    // pendingIncomingEvent so a future arming starts fresh.
    const idleBtn = screen.getByTestId("idle-trigger")
    await act(async () => {
      idleBtn.click()
    })

    // After idle, the latest props must show pendingIncomingEvent = null.
    await waitFor(() => {
      const props = runtimeRenderSpy.mock.calls.at(-1)?.[0] as Record<
        string,
        unknown
      >
      expect(props.pendingIncomingEvent).toBeNull()
    })
  })
})
