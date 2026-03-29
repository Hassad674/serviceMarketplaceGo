"use client"

import { useState, useRef, useCallback, useEffect } from "react"
import { Room, RoomEvent, RemoteTrack, Track, RemoteTrackPublication, RemoteParticipant } from "livekit-client"
import {
  initiateCall,
  acceptCall as acceptCallApi,
  declineCall as declineCallApi,
  endCall as endCallApi,
} from "../api/call-api"
import type { CallState, CallType, CallViewMode, ActiveCall, IncomingCall, CallEventPayload } from "../types"

const RING_TIMEOUT_MS = 30_000

export function useCall() {
  const [state, setState] = useState<CallState>("idle")
  const [activeCall, setActiveCall] = useState<ActiveCall | null>(null)
  const [incomingCall, setIncomingCall] = useState<IncomingCall | null>(null)
  const [isMuted, setIsMuted] = useState(false)
  const [isCameraOff, setIsCameraOff] = useState(false)
  const [duration, setDuration] = useState(0)
  const [viewMode, setViewMode] = useState<CallViewMode>("pip")
  const [room, setRoom] = useState<Room | null>(null)

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
    // Remove any attached LiveKit audio elements from the DOM
    document
      .querySelectorAll("audio[data-livekit-audio]")
      .forEach((el) => el.remove())
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
    setIsCameraOff(false)
    setDuration(0)
    setViewMode("pip")
    setRoom(null)
  }, [clearTimers, disconnectRoom])

  const startDurationTimer = useCallback(() => {
    if (durationTimerRef.current) {
      clearInterval(durationTimerRef.current)
      durationTimerRef.current = null
    }
    setDuration(0)
    const start = Date.now()
    durationTimerRef.current = setInterval(() => {
      const secs = Math.floor((Date.now() - start) / 1000)
      setDuration(secs)
    }, 1000)
  }, [])

  const connectToRoom = useCallback(async (token: string, callType: CallType = "audio") => {
    const wsUrl = process.env.NEXT_PUBLIC_LIVEKIT_URL || ""
    if (!wsUrl) {
      console.warn("[Call] NEXT_PUBLIC_LIVEKIT_URL is not set, skipping room connection")
      return
    }

    try {
      const newRoom = new Room()
      roomRef.current = newRoom
      setRoom(newRoom)

      newRoom.on(
        RoomEvent.TrackSubscribed,
        (track: RemoteTrack, _pub: RemoteTrackPublication, _participant: RemoteParticipant) => {
          if (track.kind === Track.Kind.Audio) {
            const el = track.attach()
            el.setAttribute("data-livekit-audio", "true")
            document.body.appendChild(el)
          }
        },
      )

      newRoom.on(
        RoomEvent.TrackUnsubscribed,
        (track: RemoteTrack) => {
          track.detach().forEach((el) => el.remove())
        },
      )

      // Do NOT auto-cleanup on Disconnected. LiveKit handles reconnection
      // internally. The call UI stays visible so the user can always hang up.
      // Cleanup happens only via explicit actions:
      //   - User clicks hangup → doHangup() → endCallApi + cleanup
      //   - Remote hangup → WS call_ended event → cleanup
      //   - Ring timeout (30s) → doHangup()

      await newRoom.connect(wsUrl, token)
      await newRoom.localParticipant.setMicrophoneEnabled(true)

      if (callType === "video") {
        try {
          await newRoom.localParticipant.setCameraEnabled(true)
        } catch (camErr) {
          console.warn("[Call] Camera failed, continuing without video:", camErr)
          setIsCameraOff(true)
        }
      }
    } catch (err) {
      console.error("[Call] Failed to connect to LiveKit room:", err)
    }
  }, [])

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
    callType: CallType = "audio",
  ) => {
    if (stateRef.current !== "idle") return
    setState("ringing_outgoing")

    try {
      const result = await initiateCall(conversationId, recipientId, callType)
      const call: ActiveCall = {
        callId: result.call_id,
        conversationId,
        roomName: result.room_name,
        token: result.token,
        callType,
        startedAt: null,
      }
      setActiveCall(call)

      // Connect to LiveKit immediately for the initiator so the
      // recipient can hear audio as soon as they accept.
      connectToRoom(result.token, callType)

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

      await connectToRoom(result.token, incomingCall.callType)
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

  const toggleCamera = useCallback(() => {
    if (!roomRef.current) return
    const newOff = !isCameraOff
    roomRef.current.localParticipant.setCameraEnabled(!newOff)
    setIsCameraOff(newOff)
  }, [isCameraOff])

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
        if (currentState !== "ringing_outgoing") return
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
      case "call_ended": {
        const currentCallId = activeCallRef.current?.callId
        if (payload.call_id === currentCallId || currentState === "ringing_incoming") {
          cleanup()
        }
        break
      }
    }
  }, [cleanup, startDurationTimer])

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      if (ringTimerRef.current) clearTimeout(ringTimerRef.current)
      if (durationTimerRef.current) clearInterval(durationTimerRef.current)
      if (roomRef.current) roomRef.current.disconnect()
      document
        .querySelectorAll("audio[data-livekit-audio]")
        .forEach((el) => el.remove())
    }
  }, [])

  return {
    state,
    activeCall,
    incomingCall,
    isMuted,
    isCameraOff,
    duration,
    viewMode,
    room,
    startCall,
    acceptIncoming,
    declineIncoming,
    hangup: doHangup,
    toggleMute,
    toggleCamera,
    setViewMode,
    handleCallEvent,
  }
}
