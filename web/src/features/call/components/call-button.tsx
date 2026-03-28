"use client"

import { Phone, Video } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import type { CallType } from "../types"

interface CallButtonProps {
  isOnline: boolean
  onStartCall: (type: CallType) => void
}

export function CallButton({ isOnline, onStartCall }: CallButtonProps) {
  const t = useTranslations("call")

  const btnClass = cn(
    "rounded-xl p-2 transition-all duration-200 active:scale-[0.98]",
    isOnline
      ? "text-emerald-600 hover:bg-emerald-50 dark:text-emerald-400 dark:hover:bg-emerald-500/10"
      : "cursor-not-allowed text-gray-300 dark:text-gray-600",
  )

  return (
    <div className="hidden items-center gap-1 sm:flex">
      <button
        type="button"
        onClick={() => onStartCall("audio")}
        disabled={!isOnline}
        className={btnClass}
        aria-label={isOnline ? t("startAudioCall") : t("recipientOffline")}
        title={isOnline ? t("startAudioCall") : t("recipientOffline")}
      >
        <Phone className="h-4 w-4" strokeWidth={1.5} />
      </button>
      <button
        type="button"
        onClick={() => onStartCall("video")}
        disabled={!isOnline}
        className={btnClass}
        aria-label={isOnline ? t("startVideoCall") : t("recipientOffline")}
        title={isOnline ? t("startVideoCall") : t("recipientOffline")}
      >
        <Video className="h-4 w-4" strokeWidth={1.5} />
      </button>
    </div>
  )
}
