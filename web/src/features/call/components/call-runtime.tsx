"use client"

import { useEffect, useRef } from "react"
import { useCall } from "../hooks/use-call"
import { IncomingCallOverlay } from "./incoming-call-overlay"
import { CallOverlay } from "./call-overlay"
import type { CallEventPayload, CallType } from "../types"

// CallRuntime is the LiveKit-aware layer of the call feature. It is
// the single component that imports `useCall` (and therefore
// `livekit-client`); it must be loaded ONLY through the dynamic
// import in `call-slot.tsx`. Direct imports defeat the entire
// purpose of the split and pull the LiveKit chunk back into the main
// bundle.
//
// Responsibilities:
//   - mount `useCall` (one-time per session, after activation)
//   - relay WS `call_event` frames into `useCall.handleCallEvent`
//   - replay any pending startCall / incoming event captured by the
//     slot before this component existed
//   - render the existing IncomingCallOverlay / CallOverlay UIs
//     (kept exactly as before for zero UX regression)
//
// IncomingCallOverlay / CallOverlay are imported statically here
// because this entire file is itself loaded via `dynamic()` in
// `call-slot.tsx`. Adding extra `dynamic()` boundaries inside the
// runtime would cost extra round-trips for no benefit.

interface PendingStartCall {
  conversationId: string
  recipientId: string
  recipientName: string
  callType: CallType
}

interface CallRuntimeProps {
  pendingStartCall: PendingStartCall | null
  pendingIncomingEvent: CallEventPayload | null
  onIdle: () => void
  registerCallEventHandler: (
    handler: ((payload: CallEventPayload) => void) | undefined,
  ) => void
}

export function CallRuntime({
  pendingStartCall,
  pendingIncomingEvent,
  onIdle,
  registerCallEventHandler,
}: CallRuntimeProps) {
  const call = useCall()
  const recipientNameRef = useRef("")
  const consumedStartRef = useRef(false)
  const consumedIncomingRef = useRef(false)

  // Wire the LiveKit-aware call event handler through the global WS
  // bridge. This replaces the slot's pre-mount handler so subsequent
  // events are routed straight to `useCall`.
  useEffect(() => {
    registerCallEventHandler(call.handleCallEvent)
    return () => {
      registerCallEventHandler(undefined)
    }
  }, [registerCallEventHandler, call.handleCallEvent])

  // Replay the start-call request that was queued before this
  // component mounted. We use a ref-based one-shot guard so a
  // repeated render doesn't re-trigger the same call.
  useEffect(() => {
    if (!pendingStartCall || consumedStartRef.current) return
    consumedStartRef.current = true
    recipientNameRef.current = pendingStartCall.recipientName
    void call.startCall(
      pendingStartCall.conversationId,
      pendingStartCall.recipientId,
      pendingStartCall.callType,
    )
    onIdle()
  }, [pendingStartCall, call.startCall, onIdle, call])

  // Replay the incoming-call frame captured by the slot before
  // mount. Same one-shot semantics as above.
  useEffect(() => {
    if (!pendingIncomingEvent || consumedIncomingRef.current) return
    consumedIncomingRef.current = true
    call.handleCallEvent(pendingIncomingEvent)
    onIdle()
  }, [pendingIncomingEvent, call.handleCallEvent, onIdle, call])

  // Update the cached recipient name when an outgoing call starts via
  // the live CallContext (e.g. from an already-mounted runtime).
  // The slot's pre-mount path captures it through pendingStartCall;
  // here we keep it in sync for ringing_outgoing renders.
  useEffect(() => {
    if (call.state === "idle") {
      recipientNameRef.current = ""
    }
  }, [call.state])

  return (
    <>
      {call.state === "ringing_incoming" && call.incomingCall ? (
        <IncomingCallOverlay
          call={call.incomingCall}
          onAccept={call.acceptIncoming}
          onDecline={call.declineIncoming}
        />
      ) : null}

      {(call.state === "active" || call.state === "ringing_outgoing") ? (
        <CallOverlay
          state={call.state}
          callType={call.callType}
          recipientName={recipientNameRef.current}
          duration={call.duration}
          isMuted={call.isMuted}
          isCameraOff={call.isCameraOff}
          viewMode={call.viewMode}
          room={call.room}
          onToggleMute={call.toggleMute}
          onToggleCamera={call.toggleCamera}
          onHangup={call.hangup}
          onSetViewMode={call.setViewMode}
        />
      ) : null}
    </>
  )
}
