"use client"

import { ArrowLeft, Wifi, WifiOff } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import type { Conversation } from "../types"
import { TypingIndicator } from "./typing-indicator"

interface ConversationHeaderProps {
  conversation: Conversation
  onBack?: () => void
  typingUserName?: string
  isConnected: boolean
}

export function ConversationHeader({
  conversation,
  onBack,
  typingUserName,
  isConnected,
}: ConversationHeaderProps) {
  const t = useTranslations("messaging")

  const initials = conversation.other_user_name
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
          aria-label={t("back")}
        >
          <ArrowLeft className="h-5 w-5" strokeWidth={1.5} />
        </button>
      )}

      {/* Avatar */}
      <div className="relative shrink-0">
        {conversation.other_photo_url ? (
          // eslint-disable-next-line @next/next/no-img-element
          <img
            src={conversation.other_photo_url}
            alt={conversation.other_user_name}
            className="h-10 w-10 rounded-full object-cover"
          />
        ) : (
          <div className="flex h-10 w-10 items-center justify-center rounded-full bg-gradient-to-br from-rose-500 to-purple-600 text-sm font-semibold text-white">
            {initials}
          </div>
        )}
        {conversation.online && (
          <span className="absolute bottom-0 right-0 h-3 w-3 rounded-full border-2 border-white bg-emerald-500 dark:border-gray-900" />
        )}
      </div>

      {/* Name and status */}
      <div className="min-w-0 flex-1">
        <p className="truncate text-sm font-semibold text-gray-900 dark:text-white">
          {conversation.other_user_name}
        </p>
        {typingUserName ? (
          <TypingIndicator userName={typingUserName} />
        ) : (
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
        )}
      </div>

      {/* Connection indicator */}
      <div className="flex items-center gap-1">
        {isConnected ? (
          <Wifi
            className="h-4 w-4 text-emerald-500"
            strokeWidth={1.5}
            aria-label={t("connected")}
          />
        ) : (
          <WifiOff
            className="h-4 w-4 text-gray-400 dark:text-gray-500"
            strokeWidth={1.5}
            aria-label={t("reconnecting")}
          />
        )}
      </div>
    </div>
  )
}
