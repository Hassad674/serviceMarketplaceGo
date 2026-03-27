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

  // Keep refs in sync
  useEffect(() => { activeCallRef.current = activeCall }, [activeCall])
  useEffect(() => { durationRef.current = duration }, [duration])

  const cleanup = useCallback(() => {
    if (ringTimerRef.current) {
      clearTimeout(ringTimerRef.current)
      ringTimerRef.current = null
    }
    if (durationTimerRef.current) {
      clearInterval(durationTimerRef.current)
      durationTimerRef.current = null
    }
    if (roomRef.current) {
      roomRef.current.disconnect()
      roomRef.current = null
    }
    setState("idle")
    setActiveCall(null)
    setIncomingCall(null)
    setIsMuted(false)
    setDuration(0)
  }, [])

  const startDurationTimer = useCallback(() => {
    const start = Date.now()
    durationTimerRef.current = setInterval(() => {
      const secs = Math.floor((Date.now() - start) / 1000)
      setDuration(secs)
    }, 1000)
  }, [])

  const connectToRoom = useCallback(async (token: string) => {
    const room = new Room()
    roomRef.current = room

    room.on(RoomEvent.TrackSubscribed, (_track: RemoteTrack) => {
      // Remote audio track auto-plays via WebAudio
    })

    room.on(RoomEvent.Disconnected, () => {
      cleanup()
    })

    const wsUrl = process.env.NEXT_PUBLIC_LIVEKIT_URL || ""
    await room.connect(wsUrl, token)
    await room.localParticipant.setMicrophoneEnabled(true)
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

  const startCall = useCallback(async (conversationId: string, recipientId: string) => {
    if (state !== "idle") return
    setState("ringing_outgoing")

    try {
      const result = await initiateCall(conversationId, recipientId, "audio")
      setActiveCall({
        callId: result.call_id,
        conversationId,
        roomName: result.room_name,
        token: result.token,
        callType: "audio",
        startedAt: null,
      })

      await connectToRoom(result.token)

      ringTimerRef.current = setTimeout(() => {
        doHangup()
      }, RING_TIMEOUT_MS)
    } catch {
      cleanup()
    }
  }, [state, connectToRoom, cleanup, doHangup])

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
      startDurationTimer()

      await connectToRoom(result.token)
    } catch {
      cleanup()
    }
  }, [incomingCall, connectToRoom, cleanup, startDurationTimer])

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
    switch (payload.event) {
      case "call_incoming":
        if (state !== "idle") return
        setIncomingCall({
          callId: payload.call_id,
          conversationId: payload.conversation_id,
          initiatorId: payload.initiator_id,
          initiatorName: "",
          callType: payload.call_type,
        })
        setState("ringing_incoming")
        ringTimerRef.current = setTimeout(() => {
          declineIncoming()
        }, RING_TIMEOUT_MS)
        break

      case "call_accepted":
        if (ringTimerRef.current) {
          clearTimeout(ringTimerRef.current)
          ringTimerRef.current = null
        }
        setState("active")
        if (activeCall) {
          setActiveCall({ ...activeCall, startedAt: Date.now() })
        }
        startDurationTimer()
        break

      case "call_declined":
      case "call_ended":
        cleanup()
        break
    }
  }, [state, activeCall, cleanup, declineIncoming, startDurationTimer])

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
