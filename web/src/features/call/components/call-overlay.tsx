"use client"

import { useCallback } from "react"
import type { Room } from "livekit-client"
import { useVideoTracks } from "../hooks/use-video-tracks"
import { PipCallOverlay } from "./pip-call-overlay"
import { FullscreenCallOverlay } from "./fullscreen-call-overlay"
import type { CallState, CallType, CallViewMode } from "../types"

interface CallOverlayProps {
  state: CallState
  callType: CallType
  recipientName: string
  duration: number
  isMuted: boolean
  isCameraOff: boolean
  viewMode: CallViewMode
  room: Room | null
  onToggleMute: () => void
  onToggleCamera: () => void
  onHangup: () => void
  onSetViewMode: (mode: CallViewMode) => void
}

export function CallOverlay({
  state,
  callType,
  recipientName,
  duration,
  isMuted,
  isCameraOff,
  viewMode,
  room,
  onToggleMute,
  onToggleCamera,
  onHangup,
  onSetViewMode,
}: CallOverlayProps) {
  const { remoteVideoTrack, localVideoTrack } = useVideoTracks(room, callType)

  const switchToFullscreen = useCallback(() => {
    onSetViewMode("fullscreen")
  }, [onSetViewMode])

  const switchToPip = useCallback(() => {
    onSetViewMode("pip")
  }, [onSetViewMode])

  if (viewMode === "fullscreen") {
    return (
      <FullscreenCallOverlay
        callType={callType}
        recipientName={recipientName}
        duration={duration}
        isMuted={isMuted}
        isCameraOff={isCameraOff}
        remoteVideoTrack={remoteVideoTrack}
        localVideoTrack={localVideoTrack}
        onToggleMute={onToggleMute}
        onToggleCamera={onToggleCamera}
        onHangup={onHangup}
        onMinimize={switchToPip}
      />
    )
  }

  return (
    <PipCallOverlay
      state={state}
      callType={callType}
      recipientName={recipientName}
      duration={duration}
      isMuted={isMuted}
      isCameraOff={isCameraOff}
      remoteVideoTrack={remoteVideoTrack}
      localVideoTrack={localVideoTrack}
      onToggleMute={onToggleMute}
      onToggleCamera={onToggleCamera}
      onHangup={onHangup}
      onSwitchToFullscreen={switchToFullscreen}
    />
  )
}
