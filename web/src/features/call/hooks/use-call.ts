"use client"

import { useState, useRef, useCallback, useEffect } from "react"
import { Room, RoomEvent, RemoteTrack } from "livekit-client"
import {
  initiateCall,
  acceptCall as acceptCallApi,
  declineCall as declineCallApi,
  endCall as endCallApi,
} from "../api/call-api"
import type { CallState, ActiveCall, IncomingCall, CallEventPayload } from "../types"

const RING_TIMEOUT_MS = 30_000

export function useCall() {
  const [state, setState] = useState<CallState>("idle")
  const [activeCall, setActiveCall] = useState<ActiveCall | null>(null)
  const [incomingCall, setIncomingCall] = useState<IncomingCall | null>(null)
  const [isMuted, setIsMuted] = useState(false)
  const [duration, setDuration] = useState(0)

  const roomRef = useRef<Room | null>(null)
  const ringTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const durationTimerRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const activeCallRef = useRef<ActiveCall | null>(null)
  const durationRef = useRef(0)
  const stateRef = useRef<CallState>("idle")

  // Keep refs in sync
  useEffect(() => { activeCallRef.current = activeCall }, [activeCall])
  useEffect(() => { durationRef.current = duration }, [duration])
  useEffect(() => { stateRef.current = state }, [state])

  const disconnectRoom = useCallback(() => {
    if (roomRef.current) {
      roomRef.current.disconnect()
      roomRef.current = null
    }
  }, [])

  const clearTimers = useCallback(() => {
    if (ringTimerRef.current) {
      clearTimeout(ringTimerRef.current)
      ringTimerRef.current = null
    }
    if (durationTimerRef.current) {
      clearInterval(durationTimerRef.current)
      durationTimerRef.current = null
    }
  }, [])

  const cleanup = useCallback(() => {
    clearTimers()
    disconnectRoom()
    setState("idle")
    setActiveCall(null)
    setIncomingCall(null)
    setIsMuted(false)
    setDuration(0)
  }, [clearTimers, disconnectRoom])

  const startDurationTimer = useCallback(() => {
    const start = Date.now()
    durationTimerRef.current = setInterval(() => {
      const secs = Math.floor((Date.now() - start) / 1000)
      setDuration(secs)
    }, 1000)
  }, [])

  const connectToRoom = useCallback(async (token: string) => {
    const wsUrl = process.env.NEXT_PUBLIC_LIVEKIT_URL || ""
    if (!wsUrl) {
      console.warn("[Call] NEXT_PUBLIC_LIVEKIT_URL is not set, skipping room connection")
      return
    }

    try {
      const room = new Room()
      roomRef.current = room

      room.on(RoomEvent.TrackSubscribed, (_track: RemoteTrack) => {
        // Remote audio track auto-plays via WebAudio
      })

      room.on(RoomEvent.Disconnected, () => {
        // Only cleanup if we are in an active call -- a disconnect during
        // ringing is expected (no room yet for the recipient)
        if (stateRef.current === "active") {
          cleanup()
        }
      })

      await room.connect(wsUrl, token)
      await room.localParticipant.setMicrophoneEnabled(true)
    } catch (err) {
      console.error("[Call] Failed to connect to LiveKit room:", err)
      // Do NOT cleanup signaling state -- the call can still function
      // at the signaling level even without media.
    }
  }, [cleanup])

  const doHangup = useCallback(async () => {
    const call = activeCallRef.current
    if (!call) {
      cleanup()
      return
    }
    try {
      await endCallApi(call.callId, durationRef.current)
    } catch {
      // Ignore errors on hangup
    }
    cleanup()
  }, [cleanup])

  const startCall = useCallback(async (
    conversationId: string,
    recipientId: string,
  ) => {
    if (stateRef.current !== "idle") return
    setState("ringing_outgoing")

    try {
      const result = await initiateCall(conversationId, recipientId, "audio")
      const call: ActiveCall = {
        callId: result.call_id,
        conversationId,
        roomName: result.room_name,
        token: result.token,
        callType: "audio",
        startedAt: null,
      }
      setActiveCall(call)

      // Connect to LiveKit immediately for the initiator so the
      // recipient can hear audio as soon as they accept.
      // This is fire-and-forget -- if it fails, the ringing UI stays.
      connectToRoom(result.token)

      ringTimerRef.current = setTimeout(() => {
        doHangup()
      }, RING_TIMEOUT_MS)
    } catch (err) {
      console.error("[Call] Failed to initiate call:", err)
      cleanup()
    }
  }, [connectToRoom, cleanup, doHangup])

  const acceptIncoming = useCallback(async () => {
    if (!incomingCall) return

    try {
      const result = await acceptCallApi(incomingCall.callId)
      setActiveCall({
        callId: incomingCall.callId,
        conversationId: incomingCall.conversationId,
        roomName: result.room_name,
        token: result.token,
        callType: incomingCall.callType,
        startedAt: Date.now(),
      })
      setIncomingCall(null)
      setState("active")
      clearTimers()
      startDurationTimer()

      await connectToRoom(result.token)
    } catch (err) {
      console.error("[Call] Failed to accept call:", err)
      cleanup()
    }
  }, [incomingCall, connectToRoom, cleanup, startDurationTimer, clearTimers])

  const declineIncoming = useCallback(async () => {
    if (!incomingCall) return
    try {
      await declineCallApi(incomingCall.callId)
    } catch {
      // Ignore errors on decline
    }
    cleanup()
  }, [incomingCall, cleanup])

  const toggleMute = useCallback(() => {
    if (!roomRef.current) return
    const newMuted = !isMuted
    roomRef.current.localParticipant.setMicrophoneEnabled(!newMuted)
    setIsMuted(newMuted)
  }, [isMuted])

  // Handle WS call events
  const handleCallEvent = useCallback((payload: CallEventPayload) => {
    const currentState = stateRef.current

    switch (payload.event) {
      case "call_incoming":
        if (currentState !== "idle") return
        setIncomingCall({
          callId: payload.call_id,
          conversationId: payload.conversation_id,
          initiatorId: payload.initiator_id,
          initiatorName: payload.initiator_name || "",
          callType: payload.call_type,
        })
        setState("ringing_incoming")
        ringTimerRef.current = setTimeout(() => {
          // Auto-decline after timeout
          if (stateRef.current === "ringing_incoming") {
            declineCallApi(payload.call_id).catch(() => {})
            cleanup()
          }
        }, RING_TIMEOUT_MS)
        break

      case "call_accepted":
        if (ringTimerRef.current) {
          clearTimeout(ringTimerRef.current)
          ringTimerRef.current = null
        }
        setState("active")
        setActiveCall((prev) =>
          prev ? { ...prev, startedAt: Date.now() } : prev,
        )
        startDurationTimer()
        break

      case "call_declined":
      case "call_ended":
        cleanup()
        break
    }
  }, [cleanup, startDurationTimer])

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      if (ringTimerRef.current) clearTimeout(ringTimerRef.current)
      if (durationTimerRef.current) clearInterval(durationTimerRef.current)
      if (roomRef.current) roomRef.current.disconnect()
    }
  }, [])

  return {
    state,
    activeCall,
    incomingCall,
    isMuted,
    duration,
    startCall,
    acceptIncoming,
    declineIncoming,
    hangup: doHangup,
    toggleMute,
    handleCallEvent,
  }
}
