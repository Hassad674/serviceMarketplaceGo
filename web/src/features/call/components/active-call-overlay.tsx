"use client"

import { PhoneOff, Mic, MicOff, Phone } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import type { CallState } from "../types"

interface ActiveCallOverlayProps {
  state: CallState
  recipientName: string
  duration: number
  isMuted: boolean
  onToggleMute: () => void
  onHangup: () => void
}

function formatDuration(seconds: number): string {
  const m = Math.floor(seconds / 60)
    .toString()
    .padStart(2, "0")
  const s = (seconds % 60).toString().padStart(2, "0")
  return `${m}:${s}`
}

export function ActiveCallOverlay({
  state,
  recipientName,
  duration,
  isMuted,
  onToggleMute,
  onHangup,
}: ActiveCallOverlayProps) {
  const t = useTranslations("call")

  const initials = recipientName
    .split(" ")
    .map((w) => w.charAt(0))
    .join("")
    .slice(0, 2)
    .toUpperCase()

  const isRinging = state === "ringing_outgoing"

  return (
    <div className="fixed bottom-6 right-6 z-[150] w-72 overflow-hidden rounded-2xl border border-gray-200 bg-gray-900 text-white shadow-2xl dark:border-gray-700">
      {/* Header */}
      <div className="flex items-center gap-3 p-4">
        <div className="flex h-10 w-10 items-center justify-center rounded-full bg-gradient-to-br from-rose-500 to-purple-600 text-sm font-semibold">
          {initials}
        </div>
        <div className="min-w-0 flex-1">
          <p className="truncate text-sm font-semibold">{recipientName}</p>
          <p className="text-xs text-gray-400">
            {isRinging ? (
              <span className="flex items-center gap-1">
                <Phone className="h-3 w-3 animate-pulse" />
                {t("calling")}
              </span>
            ) : (
              <span className="font-mono">{formatDuration(duration)}</span>
            )}
          </p>
        </div>
      </div>

      {/* Controls */}
      <div className="flex items-center justify-center gap-4 border-t border-gray-800 px-4 py-3">
        <button
          type="button"
          onClick={onToggleMute}
          className={cn(
            "flex h-10 w-10 items-center justify-center rounded-full transition-all duration-200",
            isMuted
              ? "bg-red-500/20 text-red-400 hover:bg-red-500/30"
              : "bg-white/10 text-white hover:bg-white/20",
          )}
          aria-label={isMuted ? t("unmute") : t("mute")}
        >
          {isMuted ? <MicOff className="h-5 w-5" /> : <Mic className="h-5 w-5" />}
        </button>

        <button
          type="button"
          onClick={onHangup}
          className={cn(
            "flex h-12 w-12 items-center justify-center rounded-full",
            "bg-red-500 text-white transition-all duration-200",
            "hover:bg-red-600 hover:shadow-lg",
            "active:scale-[0.95]",
          )}
          aria-label={t("hangup")}
        >
          <PhoneOff className="h-5 w-5" />
        </button>
      </div>
    </div>
  )
}
