"use client"

import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ComponentType,
} from "react"
import dynamic from "next/dynamic"
import { CallContext } from "@/shared/hooks/use-call-context"
import type { CallEventPayload, CallType } from "../types"

// CallSlot is the perf-conscious entry point for the call feature.
//
// **Why this exists**: `useCall` imports `livekit-client`, a 1.3 MB
// runtime dependency. Wiring `useCall` directly into `DashboardShell`
// (the previous shape) means every authenticated dashboard user
// downloads LiveKit even when they never start or receive a call.
// CallSlot replaces that direct hook call with a thin React boundary
// that:
//
//  1. Registers itself with `useGlobalWS` via the `onCallEvent` prop —
//     the global WebSocket is the only path through which incoming
//     `call_incoming` frames can reach the page.
//  2. Exposes a stable `startCall` callback through `CallContext` so
//     downstream features (`messaging-page.tsx`) can trigger calls
//     without first paying the LiveKit cost.
//  3. Lazily mounts `CallRuntime` (the LiveKit-using subtree) on
//     first activation. Activation triggers when EITHER:
//       - a WebSocket `call_incoming` frame arrives, OR
//       - the user invokes `startCall(...)` from the messaging UI.
//
// Once activated, the slot stays armed for the lifetime of the
// session. The runtime owns its own lifecycle (timeouts, room
// cleanup) and keeps consuming WS events through the prop bridge.

interface CallSlotProps {
  registerCallEventHandler: (handler: ((payload: CallEventPayload) => void) | undefined) => void
  /**
   * Optional descendants that need read access to the CallContext.
   * In the dashboard shell, the page tree (`<main>{children}</main>`)
   * lives outside this slot. To keep `useCallContext()` working in
   * downstream features (`messaging-page.tsx` calls startCall), the
   * shell wraps its main tree as children of CallSlot — see
   * `dashboard-shell.tsx`.
   */
  children?: React.ReactNode
}

type StartCallArgs = {
  conversationId: string
  recipientId: string
  recipientName: string
  callType: CallType
}

interface CallRuntimeProps {
  pendingStartCall: StartCallArgs | null
  pendingIncomingEvent: CallEventPayload | null
  onIdle: () => void
  registerCallEventHandler: (handler: ((payload: CallEventPayload) => void) | undefined) => void
}

// dynamic() with ssr:false is the Next.js documented split-point: the
// import promise only resolves on the client at first render of this
// component, which itself only renders once `armed` flips to true.
// `loading: () => null` avoids any visible UI flash because the call
// overlays handle their own appearance once mounted.
const CallRuntime = dynamic<CallRuntimeProps>(
  () =>
    import("./call-runtime").then((m) => ({ default: m.CallRuntime })),
  { ssr: false, loading: () => null },
)

export function CallSlot({ registerCallEventHandler, children }: CallSlotProps) {
  const [armed, setArmed] = useState(false)

  // pendingStartCall / pendingIncomingEvent are one-shot handoffs that
  // get reset to null once `CallRuntime` consumes them. Without this
  // shape the very first call attempt would race the dynamic import.
  const [pendingStartCall, setPendingStartCall] = useState<StartCallArgs | null>(null)
  const [pendingIncomingEvent, setPendingIncomingEvent] = useState<CallEventPayload | null>(null)

  const armedRef = useRef(false)
  useEffect(() => {
    armedRef.current = armed
  }, [armed])

  // Pre-mount handler: this is the function we hand to the global WS.
  // Until the runtime loads, it captures the FIRST call_incoming
  // event and arms the slot. After arming, the runtime swaps in its
  // own handler via `registerCallEventHandler`.
  const preMountHandler = useCallback((payload: CallEventPayload) => {
    if (armedRef.current) return
    if (payload.event !== "call_incoming") return
    setPendingIncomingEvent(payload)
    setArmed(true)
  }, [])

  useEffect(() => {
    registerCallEventHandler(preMountHandler)
    return () => {
      registerCallEventHandler(undefined)
    }
  }, [registerCallEventHandler, preMountHandler])

  const startCall = useCallback(
    async (
      conversationId: string,
      recipientId: string,
      recipientName?: string,
      callType?: "audio" | "video",
    ) => {
      const next: StartCallArgs = {
        conversationId,
        recipientId,
        recipientName: recipientName ?? "",
        callType: callType ?? "audio",
      }
      setPendingStartCall(next)
      setArmed(true)
    },
    [],
  )

  const callContextValue = useMemo(() => ({ startCall }), [startCall])

  const handleIdle = useCallback(() => {
    setPendingStartCall(null)
    setPendingIncomingEvent(null)
  }, [])

  return (
    <CallContext.Provider value={callContextValue}>
      {children}
      {armed ? (
        <CallRuntime
          pendingStartCall={pendingStartCall}
          pendingIncomingEvent={pendingIncomingEvent}
          onIdle={handleIdle}
          registerCallEventHandler={registerCallEventHandler}
        />
      ) : null}
    </CallContext.Provider>
  )
}

// Test seam: re-exported so unit tests can swap the dynamic component
// without touching the production wiring.
export const __testInternals: { CallRuntime: ComponentType<CallRuntimeProps> } = {
  CallRuntime,
}
