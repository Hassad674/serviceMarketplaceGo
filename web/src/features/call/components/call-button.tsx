"use client"

import { Phone } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"

interface CallButtonProps {
  isOnline: boolean
  onClick: () => void
}

export function CallButton({ isOnline, onClick }: CallButtonProps) {
  const t = useTranslations("call")

  return (
    <button
      type="button"
      onClick={onClick}
      disabled={!isOnline}
      className={cn(
        "hidden items-center gap-2 rounded-xl px-4 py-2 text-sm font-medium sm:flex",
        "transition-all duration-200 active:scale-[0.98]",
        isOnline
          ? "text-emerald-600 hover:bg-emerald-50 dark:text-emerald-400 dark:hover:bg-emerald-500/10"
          : "cursor-not-allowed text-gray-300 dark:text-gray-600",
      )}
      aria-label={isOnline ? t("startCall") : t("recipientOffline")}
      title={isOnline ? t("startCall") : t("recipientOffline")}
    >
      <Phone className="h-4 w-4" strokeWidth={1.5} />
    </button>
  )
}
