"use client"

import { Maximize2, Phone } from "lucide-react"
import { useTranslations } from "next-intl"
import type { RemoteTrack, LocalTrack } from "livekit-client"
import { cn } from "@/shared/lib/utils"
import { useDraggable } from "../hooks/use-draggable"
import { VideoRenderer } from "./video-renderer"
import { AvatarFallback } from "./avatar-fallback"
import { CallControls } from "./call-controls"
import type { CallState, CallType } from "../types"

interface PipCallOverlayProps {
  state: CallState
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
  onSwitchToFullscreen: () => void
}

function formatDuration(seconds: number): string {
  const m = Math.floor(seconds / 60).toString().padStart(2, "0")
  const s = (seconds % 60).toString().padStart(2, "0")
  return `${m}:${s}`
}

export function PipCallOverlay({
  state,
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
  onSwitchToFullscreen,
}: PipCallOverlayProps) {
  const t = useTranslations("call")
  const { position, dragHandlers } = useDraggable({
    x: typeof window !== "undefined" ? window.innerWidth - 340 : 0,
    y: typeof window !== "undefined" ? window.innerHeight - 240 : 0,
  })

  const isRinging = state === "ringing_outgoing"
  const hasRemoteVideo = callType === "video" && remoteVideoTrack != null
  const hasLocalVideo = callType === "video" && localVideoTrack
  console.log("[Video] PiP render - hasRemoteVideo:", hasRemoteVideo, "hasLocalVideo:", !!hasLocalVideo, "isRinging:", isRinging)

  return (
    <div
      className={cn(
        "fixed z-[150] overflow-hidden rounded-2xl shadow-2xl",
        "w-[280px] h-[180px] sm:w-[320px] sm:h-[200px]",
        "select-none",
      )}
      style={{ left: position.x, top: position.y }}
      {...dragHandlers}
    >
      {/* Main area — remote video or avatar */}
      <div className="relative h-full w-full">
        <MainContent
          isRinging={isRinging}
          hasRemoteVideo={!!hasRemoteVideo}
          remoteVideoTrack={remoteVideoTrack}
          recipientName={recipientName}
          t={t}
        />

        {/* Local thumbnail — bottom-left */}
        {hasLocalVideo && !isRinging && (
          <div className="absolute bottom-10 left-2 h-[60px] w-[80px] overflow-hidden rounded-lg border-2 border-white/30 shadow-md">
            <VideoRenderer track={localVideoTrack} mirror objectFit="cover" />
          </div>
        )}

        {/* Top bar: timer + fullscreen */}
        <div className="absolute inset-x-0 top-0 flex items-center justify-between bg-black/40 px-3 py-1.5 backdrop-blur-sm">
          <span className="font-mono text-xs text-white">
            {isRinging ? t("calling") : formatDuration(duration)}
          </span>
          <button
            type="button"
            onClick={onSwitchToFullscreen}
            className="rounded-md p-1 text-white/80 transition-colors hover:bg-white/20 hover:text-white"
            aria-label={t("switchToFullscreen")}
          >
            <Maximize2 className="h-3.5 w-3.5" />
          </button>
        </div>

        {/* Bottom bar: controls */}
        <div className="absolute inset-x-0 bottom-0 flex items-center justify-center bg-black/40 py-1.5 backdrop-blur-sm">
          <CallControls
            isMuted={isMuted}
            isCameraOff={isCameraOff}
            callType={callType}
            variant="compact"
            onToggleMute={onToggleMute}
            onToggleCamera={onToggleCamera}
            onHangup={onHangup}
          />
        </div>
      </div>
    </div>
  )
}

interface MainContentProps {
  isRinging: boolean
  hasRemoteVideo: boolean
  remoteVideoTrack: RemoteTrack | null
  recipientName: string
  t: ReturnType<typeof useTranslations<"call">>
}

function MainContent({
  isRinging,
  hasRemoteVideo,
  remoteVideoTrack,
  recipientName,
  t,
}: MainContentProps) {
  if (isRinging) {
    return (
      <div className="flex h-full w-full flex-col items-center justify-center bg-gray-900/80 backdrop-blur-xl">
        <AvatarFallback name={recipientName} size="md" />
        <span className="mt-2 flex items-center gap-1 text-sm text-white">
          <Phone className="h-3 w-3 animate-pulse" />
          {t("calling")}
        </span>
      </div>
    )
  }

  if (hasRemoteVideo) {
    return <VideoRenderer track={remoteVideoTrack} objectFit="cover" />
  }

  return (
    <div className="flex h-full w-full flex-col items-center justify-center bg-gray-900/80 backdrop-blur-xl">
      <AvatarFallback name={recipientName} size="md" />
      <span className="mt-1 text-xs text-gray-400">{recipientName}</span>
    </div>
  )
}
