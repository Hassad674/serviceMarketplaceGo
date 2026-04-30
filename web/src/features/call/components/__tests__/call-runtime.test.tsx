/**
 * CallRuntime tests — PERF-W-01 lazy boundary.
 *
 * The runtime is the LiveKit-aware layer; it is the ONE component
 * that imports `useCall` (and therefore livekit-client). Tests mock
 * `../hooks/use-call` so the runtime can be exercised without
 * touching the real LiveKit module.
 *
 * Contract under test:
 *   - registers its own handler with the WS bridge on mount
 *   - replays a pending startCall once
 *   - replays a pending incoming event once
 *   - renders IncomingCallOverlay only when state is ringing_incoming
 *   - renders CallOverlay when state is ringing_outgoing or active
 *   - resets the recipient cache when state returns to idle
 */

import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import { render, screen, waitFor, act } from "@testing-library/react"
import type { CallEventPayload } from "../../types"

const mockUseCall = vi.fn()

vi.mock("../../hooks/use-call", () => ({
  useCall: () => mockUseCall(),
}))

vi.mock("../incoming-call-overlay", () => ({
  IncomingCallOverlay: (props: { call: { callId: string } }) => (
    <div data-testid="incoming-overlay">{props.call.callId}</div>
  ),
}))

vi.mock("../call-overlay", () => ({
  CallOverlay: (props: { recipientName: string; state: string }) => (
    <div data-testid="call-overlay">
      {props.state}:{props.recipientName}
    </div>
  ),
}))

import { CallRuntime } from "../call-runtime"

function makeCallShape(overrides: Record<string, unknown> = {}) {
  return {
    state: "idle",
    activeCall: null,
    incomingCall: null,
    isMuted: false,
    isCameraOff: false,
    duration: 0,
    viewMode: "pip",
    room: null,
    callType: "audio",
    startCall: vi.fn().mockResolvedValue(undefined),
    acceptIncoming: vi.fn(),
    declineIncoming: vi.fn(),
    hangup: vi.fn(),
    toggleMute: vi.fn(),
    toggleCamera: vi.fn(),
    setViewMode: vi.fn(),
    handleCallEvent: vi.fn(),
    ...overrides,
  }
}

beforeEach(() => {
  mockUseCall.mockReset()
})

afterEach(() => {
  vi.restoreAllMocks()
})

describe("CallRuntime — lazy LiveKit layer", () => {
  it("registers its own WS handler on mount and unregisters on unmount", () => {
    const register = vi.fn()
    const handleCallEvent = vi.fn()
    mockUseCall.mockReturnValue(makeCallShape({ handleCallEvent }))

    const { unmount } = render(
      <CallRuntime
        pendingStartCall={null}
        pendingIncomingEvent={null}
        onIdle={() => {}}
        registerCallEventHandler={register}
      />,
    )

    // After mount: registered the call's handler.
    expect(register).toHaveBeenCalledWith(handleCallEvent)

    unmount()

    // After unmount: must clear the handler so a stale runtime never
    // captures events.
    expect(register).toHaveBeenLastCalledWith(undefined)
  })

  it("replays a pending startCall exactly once", async () => {
    const startCall = vi.fn().mockResolvedValue(undefined)
    const onIdle = vi.fn()
    mockUseCall.mockReturnValue(makeCallShape({ startCall }))

    const { rerender } = render(
      <CallRuntime
        pendingStartCall={{
          conversationId: "c1",
          recipientId: "u1",
          recipientName: "Carol",
          callType: "video",
        }}
        pendingIncomingEvent={null}
        onIdle={onIdle}
        registerCallEventHandler={vi.fn()}
      />,
    )

    await waitFor(() => {
      expect(startCall).toHaveBeenCalledWith("c1", "u1", "video")
    })
    expect(onIdle).toHaveBeenCalled()

    // Re-render with the same prop — must NOT trigger again.
    rerender(
      <CallRuntime
        pendingStartCall={{
          conversationId: "c1",
          recipientId: "u1",
          recipientName: "Carol",
          callType: "video",
        }}
        pendingIncomingEvent={null}
        onIdle={onIdle}
        registerCallEventHandler={vi.fn()}
      />,
    )

    expect(startCall).toHaveBeenCalledTimes(1)
  })

  it("replays a pending incoming event exactly once", async () => {
    const handleCallEvent = vi.fn()
    const onIdle = vi.fn()
    mockUseCall.mockReturnValue(makeCallShape({ handleCallEvent }))

    const incoming: CallEventPayload = {
      event: "call_incoming",
      call_id: "abc",
      conversation_id: "conv-1",
      initiator_id: "user-2",
      recipient_id: "user-1",
      call_type: "audio",
    }

    const { rerender } = render(
      <CallRuntime
        pendingStartCall={null}
        pendingIncomingEvent={incoming}
        onIdle={onIdle}
        registerCallEventHandler={vi.fn()}
      />,
    )

    await waitFor(() => {
      expect(handleCallEvent).toHaveBeenCalledWith(incoming)
    })
    expect(onIdle).toHaveBeenCalled()

    rerender(
      <CallRuntime
        pendingStartCall={null}
        pendingIncomingEvent={incoming}
        onIdle={onIdle}
        registerCallEventHandler={vi.fn()}
      />,
    )

    expect(handleCallEvent).toHaveBeenCalledTimes(1)
  })

  it("renders IncomingCallOverlay only on ringing_incoming", async () => {
    mockUseCall.mockReturnValue(
      makeCallShape({
        state: "ringing_incoming",
        incomingCall: {
          callId: "x1",
          conversationId: "c",
          initiatorId: "i",
          initiatorName: "Dan",
          callType: "audio",
        },
      }),
    )

    render(
      <CallRuntime
        pendingStartCall={null}
        pendingIncomingEvent={null}
        onIdle={() => {}}
        registerCallEventHandler={vi.fn()}
      />,
    )

    expect(await screen.findByTestId("incoming-overlay")).toHaveTextContent("x1")
    expect(screen.queryByTestId("call-overlay")).toBeNull()
  })

  it("renders CallOverlay on ringing_outgoing (recipient name fills in after replay effect)", async () => {
    mockUseCall.mockReturnValue(makeCallShape({ state: "ringing_outgoing" }))

    const { rerender } = render(
      <CallRuntime
        pendingStartCall={{
          conversationId: "c",
          recipientId: "u",
          recipientName: "Eve",
          callType: "audio",
        }}
        pendingIncomingEvent={null}
        onIdle={() => {}}
        registerCallEventHandler={vi.fn()}
      />,
    )

    // The replay effect updates recipientNameRef AFTER first render.
    // Re-render to flush the cached name into the overlay (mirrors
    // the production sequence: state transitions trigger a re-render
    // that picks up the ref's current value).
    mockUseCall.mockReturnValue(makeCallShape({ state: "ringing_outgoing" }))
    rerender(
      <CallRuntime
        pendingStartCall={null}
        pendingIncomingEvent={null}
        onIdle={() => {}}
        registerCallEventHandler={vi.fn()}
      />,
    )

    expect(await screen.findByTestId("call-overlay")).toHaveTextContent(
      "ringing_outgoing:Eve",
    )
  })

  it("renders CallOverlay on active", async () => {
    mockUseCall.mockReturnValue(makeCallShape({ state: "active" }))

    render(
      <CallRuntime
        pendingStartCall={null}
        pendingIncomingEvent={null}
        onIdle={() => {}}
        registerCallEventHandler={vi.fn()}
      />,
    )

    expect(await screen.findByTestId("call-overlay")).toHaveTextContent("active:")
  })

  it("renders neither overlay on idle state", () => {
    mockUseCall.mockReturnValue(makeCallShape({ state: "idle" }))

    render(
      <CallRuntime
        pendingStartCall={null}
        pendingIncomingEvent={null}
        onIdle={() => {}}
        registerCallEventHandler={vi.fn()}
      />,
    )

    expect(screen.queryByTestId("incoming-overlay")).toBeNull()
    expect(screen.queryByTestId("call-overlay")).toBeNull()
  })

  it("resets recipient name cache when state returns to idle", async () => {
    // First render in ringing_outgoing — recipient name comes from
    // pendingStartCall.
    mockUseCall.mockReturnValue(makeCallShape({ state: "ringing_outgoing" }))

    const { rerender } = render(
      <CallRuntime
        pendingStartCall={{
          conversationId: "c",
          recipientId: "u",
          recipientName: "Frank",
          callType: "audio",
        }}
        pendingIncomingEvent={null}
        onIdle={() => {}}
        registerCallEventHandler={vi.fn()}
      />,
    )

    // Force a re-render so the cached recipientNameRef bleeds into
    // the overlay text (replay effect runs after first render).
    mockUseCall.mockReturnValue(makeCallShape({ state: "ringing_outgoing" }))
    rerender(
      <CallRuntime
        pendingStartCall={null}
        pendingIncomingEvent={null}
        onIdle={() => {}}
        registerCallEventHandler={vi.fn()}
      />,
    )
    expect(await screen.findByTestId("call-overlay")).toHaveTextContent("Frank")

    // Transition to idle — the idle effect must clear the cache.
    mockUseCall.mockReturnValue(makeCallShape({ state: "idle" }))
    rerender(
      <CallRuntime
        pendingStartCall={null}
        pendingIncomingEvent={null}
        onIdle={() => {}}
        registerCallEventHandler={vi.fn()}
      />,
    )

    // Back to ringing_outgoing — recipient should be empty because
    // no new pendingStartCall arrived.
    mockUseCall.mockReturnValue(makeCallShape({ state: "ringing_outgoing" }))
    act(() => {
      rerender(
        <CallRuntime
          pendingStartCall={null}
          pendingIncomingEvent={null}
          onIdle={() => {}}
          registerCallEventHandler={vi.fn()}
        />,
      )
    })

    expect(screen.getByTestId("call-overlay")).toHaveTextContent(
      "ringing_outgoing:",
    )
  })
})
