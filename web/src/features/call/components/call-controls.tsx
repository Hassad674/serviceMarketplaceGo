"use client"

import { Mic, MicOff, Video, VideoOff, PhoneOff } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import type { CallType } from "../types"

interface CallControlsProps {
  isMuted: boolean
  isCameraOff: boolean
  callType: CallType
  variant: "compact" | "full"
  onToggleMute: () => void
  onToggleCamera: () => void
  onHangup: () => void
}

export function CallControls({
  isMuted,
  isCameraOff,
  callType,
  variant,
  onToggleMute,
  onToggleCamera,
  onHangup,
}: CallControlsProps) {
  const t = useTranslations("call")
  const isCompact = variant === "compact"

  const btnBase = isCompact
    ? "h-8 w-8 rounded-full"
    : "h-14 w-14 rounded-full"

  const iconSize = isCompact ? "h-4 w-4" : "h-6 w-6"

  return (
    <div className="flex items-center justify-center gap-4">
      {/* Mute button */}
      <ControlButton
        active={isMuted}
        label={isMuted ? t("unmute") : t("mute")}
        sublabel={!isCompact ? (isMuted ? t("unmute") : t("mute")) : undefined}
        className={btnBase}
        activeClass="bg-red-500/20 text-red-400 hover:bg-red-500/30"
        inactiveClass="bg-white/10 text-white hover:bg-white/20"
        onClick={onToggleMute}
      >
        {isMuted ? <MicOff className={iconSize} /> : <Mic className={iconSize} />}
      </ControlButton>

      {/* Camera button — only for video calls */}
      {callType === "video" && (
        <ControlButton
          active={isCameraOff}
          label={isCameraOff ? t("cameraOn") : t("cameraOff")}
          sublabel={!isCompact ? (isCameraOff ? t("cameraOn") : t("camera")) : undefined}
          className={btnBase}
          activeClass="bg-red-500/20 text-red-400 hover:bg-red-500/30"
          inactiveClass="bg-white/10 text-white hover:bg-white/20"
          onClick={onToggleCamera}
        >
          {isCameraOff ? <VideoOff className={iconSize} /> : <Video className={iconSize} />}
        </ControlButton>
      )}

      {/* Hangup button */}
      <div className="flex flex-col items-center gap-1">
        <button
          type="button"
          onClick={onHangup}
          className={cn(
            btnBase,
            "flex items-center justify-center",
            "bg-red-500 text-white transition-all duration-200",
            "hover:bg-red-600 hover:shadow-lg active:scale-[0.95]",
            isCompact ? "h-9 w-9" : "h-14 w-14",
          )}
          aria-label={t("hangup")}
        >
          <PhoneOff className={iconSize} />
        </button>
        {!isCompact && (
          <span className="text-xs text-gray-400">{t("hangup")}</span>
        )}
      </div>
    </div>
  )
}

interface ControlButtonProps {
  active: boolean
  label: string
  sublabel?: string
  className: string
  activeClass: string
  inactiveClass: string
  onClick: () => void
  children: React.ReactNode
}

function ControlButton({
  active,
  label,
  sublabel,
  className,
  activeClass,
  inactiveClass,
  onClick,
  children,
}: ControlButtonProps) {
  return (
    <div className="flex flex-col items-center gap-1">
      <button
        type="button"
        onClick={onClick}
        className={cn(
          className,
          "flex items-center justify-center transition-all duration-200",
          active ? activeClass : inactiveClass,
        )}
        aria-label={label}
      >
        {children}
      </button>
      {sublabel && (
        <span className="text-xs text-gray-400">{sublabel}</span>
      )}
    </div>
  )
}
