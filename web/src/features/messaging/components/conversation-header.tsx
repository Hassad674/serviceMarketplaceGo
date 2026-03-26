"use client"

import { Phone, MoreVertical, ArrowLeft } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import type { Conversation } from "../types"

interface ConversationHeaderProps {
  conversation: Conversation
  onBack?: () => void
}

export function ConversationHeader({
  conversation,
  onBack,
}: ConversationHeaderProps) {
  const t = useTranslations("messaging")

  const initials = conversation.name
    .split(" ")
    .map((w) => w.charAt(0))
    .join("")
    .slice(0, 2)
    .toUpperCase()

  return (
    <div className="flex items-center gap-3 border-b border-gray-100 bg-white px-4 py-3 dark:border-gray-800 dark:bg-gray-900">
      {/* Back button (mobile only) */}
      {onBack && (
        <button
          onClick={onBack}
          className="rounded-lg p-1.5 text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-600 lg:hidden dark:hover:bg-gray-800 dark:hover:text-gray-300"
          aria-label="Back to conversations"
        >
          <ArrowLeft className="h-5 w-5" strokeWidth={1.5} />
        </button>
      )}

      {/* Avatar */}
      <div className="relative shrink-0">
        <div className="flex h-10 w-10 items-center justify-center rounded-full bg-gradient-to-br from-rose-500 to-purple-600 text-sm font-semibold text-white">
          {initials}
        </div>
        {conversation.online && (
          <span className="absolute bottom-0 right-0 h-3 w-3 rounded-full border-2 border-white bg-emerald-500 dark:border-gray-900" />
        )}
      </div>

      {/* Name and status */}
      <div className="min-w-0 flex-1">
        <p className="truncate text-sm font-semibold text-gray-900 dark:text-white">
          {conversation.name}
        </p>
        <p
          className={cn(
            "text-xs",
            conversation.online
              ? "text-emerald-600 dark:text-emerald-400"
              : "text-gray-400 dark:text-gray-500",
          )}
        >
          {conversation.online ? t("online") : t("offline")}
        </p>
      </div>

      {/* Actions */}
      <div className="flex items-center gap-1">
        <button
          className="rounded-lg p-2 text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-600 dark:hover:bg-gray-800 dark:hover:text-gray-300"
          aria-label="Phone call"
        >
          <Phone className="h-[18px] w-[18px]" strokeWidth={1.5} />
        </button>
        <button
          className="rounded-lg p-2 text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-600 dark:hover:bg-gray-800 dark:hover:text-gray-300"
          aria-label="More options"
        >
          <MoreVertical className="h-[18px] w-[18px]" strokeWidth={1.5} />
        </button>
      </div>
    </div>
  )
}
