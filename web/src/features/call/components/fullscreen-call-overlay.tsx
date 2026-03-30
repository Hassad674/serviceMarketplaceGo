"use client"

import { useState, useEffect, useCallback, useRef } from "react"
import { Minimize2 } from "lucide-react"
import { useTranslations } from "next-intl"
import type { RemoteTrack, LocalTrack } from "livekit-client"
import { cn } from "@/shared/lib/utils"
import { useDraggable } from "../hooks/use-draggable"
import { VideoRenderer } from "./video-renderer"
import { AvatarFallback } from "./avatar-fallback"
import { CallControls } from "./call-controls"
import type { CallType } from "../types"

interface FullscreenCallOverlayProps {
  callType: CallType
  recipientName: string
  duration: number
  isMuted: boolean
  isCameraOff: boolean
  remoteVideoTrack: RemoteTrack | null
  localVideoTrack: LocalTrack | null
  onToggleMute: () => void
  onToggleCamera: () => void
  onHangup: () => void
  onMinimize: () => void
}

function formatDuration(seconds: number): string {
  const m = Math.floor(seconds / 60).toString().padStart(2, "0")
  const s = (seconds % 60).toString().padStart(2, "0")
  return `${m}:${s}`
}

export function FullscreenCallOverlay({
  callType,
  recipientName,
  duration,
  isMuted,
  isCameraOff,
  remoteVideoTrack,
  localVideoTrack,
  onToggleMute,
  onToggleCamera,
  onHangup,
  onMinimize,
}: FullscreenCallOverlayProps) {
  const t = useTranslations("call")
  const [controlsVisible, setControlsVisible] = useState(true)
  const hideTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  const hasRemoteVideo = callType === "video" && remoteVideoTrack
  const hasLocalVideo = callType === "video" && localVideoTrack
  console.log("[Video] Fullscreen render - hasRemoteVideo:", !!hasRemoteVideo, "hasLocalVideo:", !!hasLocalVideo)

  const thumbPos = useDraggable({
    x: typeof window !== "undefined" ? window.innerWidth - 180 : 0,
    y: typeof window !== "undefined" ? window.innerHeight - 200 : 0,
  })

  const resetHideTimer = useCallback(() => {
    setControlsVisible(true)
    if (hideTimerRef.current) clearTimeout(hideTimerRef.current)
    hideTimerRef.current = setTimeout(() => {
      setControlsVisible(false)
    }, 3000)
  }, [])

  // Auto-hide controls after 3s
  useEffect(() => {
    resetHideTimer()
    return () => {
      if (hideTimerRef.current) clearTimeout(hideTimerRef.current)
    }
  }, [resetHideTimer])

  // ESC key listener
  useEffect(() => {
    function onKeyDown(e: KeyboardEvent) {
      if (e.key === "Escape") onMinimize()
    }
    window.addEventListener("keydown", onKeyDown)
    return () => window.removeEventListener("keydown", onKeyDown)
  }, [onMinimize])

  return (
    <div
      className="fixed inset-0 z-[200] bg-gray-950"
      onMouseMove={resetHideTimer}
      onTouchStart={resetHideTimer}
    >
      {/* Remote video / fallback */}
      <RemoteContent
        hasRemoteVideo={!!hasRemoteVideo}
        remoteVideoTrack={remoteVideoTrack}
        recipientName={recipientName}
      />

      {/* Local video thumbnail */}
      {hasLocalVideo && (
        <div
          className="absolute z-10 h-[120px] w-[160px] overflow-hidden rounded-xl border-2 border-white/20 shadow-xl"
          style={{ left: thumbPos.position.x, top: thumbPos.position.y }}
          {...thumbPos.dragHandlers}
        >
          <VideoRenderer track={localVideoTrack} mirror objectFit="cover" />
        </div>
      )}

      {/* Header */}
      <div
        className={cn(
          "absolute inset-x-0 top-0 z-20 flex items-center justify-between px-6 py-4",
          "bg-black/40 backdrop-blur-sm transition-opacity duration-300",
          controlsVisible ? "opacity-100" : "pointer-events-none opacity-0",
        )}
      >
        <div>
          <p className="text-lg font-semibold text-white">{recipientName}</p>
          <p className="font-mono text-sm text-gray-300">
            {formatDuration(duration)}
          </p>
        </div>
        <button
          type="button"
          onClick={onMinimize}
          className="rounded-lg p-2 text-white/80 transition-colors hover:bg-white/10 hover:text-white"
          aria-label={t("switchToPip")}
        >
          <Minimize2 className="h-5 w-5" />
        </button>
      </div>

      {/* Footer controls */}
      <div
        className={cn(
          "absolute inset-x-0 bottom-0 z-20 flex items-center justify-center px-6 py-6",
          "bg-black/40 backdrop-blur-sm transition-opacity duration-300",
          controlsVisible ? "opacity-100" : "pointer-events-none opacity-0",
        )}
      >
        <CallControls
          isMuted={isMuted}
          isCameraOff={isCameraOff}
          callType={callType}
          variant="full"
          onToggleMute={onToggleMute}
          onToggleCamera={onToggleCamera}
          onHangup={onHangup}
        />
      </div>
    </div>
  )
}

interface RemoteContentProps {
  hasRemoteVideo: boolean
  remoteVideoTrack: RemoteTrack | null
  recipientName: string
}

function RemoteContent({
  hasRemoteVideo,
  remoteVideoTrack,
  recipientName,
}: RemoteContentProps) {
  if (hasRemoteVideo) {
    return (
      <div className="absolute inset-0">
        <VideoRenderer track={remoteVideoTrack} objectFit="cover" />
      </div>
    )
  }

  return (
    <div className="flex h-full w-full items-center justify-center">
      <AvatarFallback name={recipientName} size="lg" />
    </div>
  )
}
