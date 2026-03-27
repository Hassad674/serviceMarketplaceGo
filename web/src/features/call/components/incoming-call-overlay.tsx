"use client"

import { useEffect, useState } from "react"
import { Phone, PhoneOff } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import type { IncomingCall } from "../types"

const RING_TIMEOUT_SEC = 30

interface IncomingCallOverlayProps {
  call: IncomingCall
  onAccept: () => void
  onDecline: () => void
}

export function IncomingCallOverlay({ call, onAccept, onDecline }: IncomingCallOverlayProps) {
  const t = useTranslations("call")
  const [elapsed, setElapsed] = useState(0)

  useEffect(() => {
    const interval = setInterval(() => {
      setElapsed((prev) => prev + 1)
    }, 1000)
    return () => clearInterval(interval)
  }, [])

  useEffect(() => {
    if (elapsed >= RING_TIMEOUT_SEC) {
      onDecline()
    }
  }, [elapsed, onDecline])

  const initials = (call.initiatorName || "?")
    .split(" ")
    .map((w) => w.charAt(0))
    .join("")
    .slice(0, 2)
    .toUpperCase()

  const progressPct = (elapsed / RING_TIMEOUT_SEC) * 100

  return (
    <div className="fixed inset-0 z-[200] flex items-center justify-center bg-black/60 backdrop-blur-sm">
      <div className="w-full max-w-sm rounded-2xl border border-gray-100 bg-white p-8 shadow-2xl dark:border-gray-800 dark:bg-gray-900">
        {/* Header */}
        <div className="mb-6 flex items-center justify-center gap-3">
          <div className="animate-pulse rounded-full bg-emerald-100 p-3 dark:bg-emerald-900/30">
            <Phone className="h-6 w-6 text-emerald-600 dark:text-emerald-400" />
          </div>
          <h2 className="text-lg font-semibold text-gray-900 dark:text-white">
            {t("incomingCall")}
          </h2>
        </div>

        {/* Caller info */}
        <div className="mb-8 flex flex-col items-center gap-4">
          <div className="flex h-20 w-20 items-center justify-center rounded-full bg-gradient-to-br from-rose-500 to-purple-600 text-xl font-bold text-white shadow-lg">
            {initials}
          </div>
          <p className="text-xl font-bold text-gray-900 dark:text-white">
            {call.initiatorName || t("unknownCaller")}
          </p>
          <p className="text-sm text-gray-500 dark:text-gray-400">
            {t("audioCall")} &middot; {elapsed}s
          </p>
        </div>

        {/* Buttons */}
        <div className="flex justify-center gap-8">
          <button
            type="button"
            onClick={onDecline}
            className={cn(
              "flex h-16 w-16 items-center justify-center rounded-full",
              "bg-red-500 text-white shadow-xl transition-all duration-200",
              "hover:scale-105 hover:bg-red-600 hover:shadow-2xl",
              "active:scale-[0.98]",
            )}
            aria-label={t("decline")}
          >
            <PhoneOff className="h-7 w-7" />
          </button>

          <button
            type="button"
            onClick={onAccept}
            className={cn(
              "flex h-16 w-16 animate-pulse items-center justify-center rounded-full",
              "bg-emerald-500 text-white shadow-xl transition-all duration-200",
              "hover:scale-105 hover:bg-emerald-600 hover:shadow-2xl",
              "active:scale-[0.98]",
            )}
            aria-label={t("accept")}
          >
            <Phone className="h-7 w-7" />
          </button>
        </div>

        {/* Labels */}
        <div className="mt-3 flex justify-center gap-8 text-xs text-gray-500 dark:text-gray-400">
          <span className="w-16 text-center">{t("decline")}</span>
          <span className="w-16 text-center">{t("accept")}</span>
        </div>

        {/* Progress bar */}
        <div className="mt-4 h-1 w-full overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700">
          <div
            className="h-full rounded-full bg-red-500 transition-all duration-1000"
            style={{ width: `${progressPct}%` }}
          />
        </div>
      </div>
    </div>
  )
}
