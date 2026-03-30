"use client"

import Image from "next/image"
import { useState, useRef, useEffect } from "react"
import { ArrowLeft, Phone, Video, Wifi, WifiOff, FileText, MoreVertical, Flag } from "lucide-react"
import { useTranslations } from "next-intl"
import { useRouter } from "@i18n/navigation"
import { cn } from "@/shared/lib/utils"
import type { Conversation } from "../types"
import { TypingIndicator } from "./typing-indicator"

interface ConversationHeaderProps {
  conversation: Conversation
  onBack?: () => void
  typingUserName?: string
  isConnected: boolean
  onStartCall?: (type: "audio" | "video") => void
  onReportUser?: () => void
}

export function ConversationHeader({
  conversation,
  onBack,
  typingUserName,
  isConnected,
  onStartCall,
  onReportUser,
}: ConversationHeaderProps) {
  const t = useTranslations("messaging")
  const tProposal = useTranslations("proposal")
  const tCall = useTranslations("call")
  const tReport = useTranslations("reporting")
  const router = useRouter()
  const [menuOpen, setMenuOpen] = useState(false)
  const menuRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
        setMenuOpen(false)
      }
    }
    if (menuOpen) {
      document.addEventListener("mousedown", handleClickOutside)
    }
    return () => document.removeEventListener("mousedown", handleClickOutside)
  }, [menuOpen])

  const initials = conversation.other_user_name
    .split(" ")
    .map((w) => w.charAt(0))
    .join("")
    .slice(0, 2)
    .toUpperCase()

  function handleStartProject() {
    router.push(`/projects/new?to=${conversation.other_user_id}&conversation=${conversation.id}`)
  }

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
          <Image
            src={conversation.other_photo_url}
            alt={conversation.other_user_name}
            width={40}
            height={40}
            className="h-10 w-10 rounded-full object-cover"
            unoptimized
          />
        ) : (
          <div className="flex h-10 w-10 items-center justify-center rounded-full bg-gradient-to-br from-rose-500 to-purple-600 text-sm font-semibold text-white">
            {initials}
          </div>
        )}
        {conversation.online && (
          <span
            className="absolute bottom-0 right-0 h-3 w-3 rounded-full border-2 border-white bg-emerald-500 dark:border-gray-900"
            aria-label={t("online")}
          >
            <span className="sr-only">{t("online")}</span>
          </span>
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

      {/* Start project button */}
      <button
        type="button"
        onClick={handleStartProject}
        className={cn(
          "hidden items-center gap-2 rounded-xl px-4 py-2 text-sm font-medium sm:flex",
          "text-rose-600 transition-all duration-200",
          "hover:bg-rose-50 dark:text-rose-400 dark:hover:bg-rose-500/10",
          "active:scale-[0.98]",
        )}
      >
        <FileText className="h-4 w-4" strokeWidth={1.5} />
        {tProposal("startProject")}
      </button>

      {/* Call buttons */}
      {onStartCall && (
        <div className="flex items-center gap-1">
          <button
            type="button"
            onClick={() => onStartCall("audio")}
            disabled={!conversation.online}
            className={cn(
              "rounded-xl p-2 transition-all duration-200",
              conversation.online
                ? "text-emerald-600 hover:bg-emerald-50 dark:text-emerald-400 dark:hover:bg-emerald-500/10"
                : "cursor-not-allowed text-gray-300 dark:text-gray-600",
            )}
            aria-label={conversation.online ? tCall("startAudioCall") : tCall("recipientOffline")}
            title={conversation.online ? tCall("startAudioCall") : tCall("recipientOffline")}
          >
            <Phone className="h-4 w-4" strokeWidth={1.5} />
          </button>
          <button
            type="button"
            onClick={() => onStartCall("video")}
            disabled={!conversation.online}
            className={cn(
              "rounded-xl p-2 transition-all duration-200",
              conversation.online
                ? "text-emerald-600 hover:bg-emerald-50 dark:text-emerald-400 dark:hover:bg-emerald-500/10"
                : "cursor-not-allowed text-gray-300 dark:text-gray-600",
            )}
            aria-label={conversation.online ? tCall("startVideoCall") : tCall("recipientOffline")}
            title={conversation.online ? tCall("startVideoCall") : tCall("recipientOffline")}
          >
            <Video className="h-4 w-4" strokeWidth={1.5} />
          </button>
        </div>
      )}

      {/* More menu (report user) */}
      {onReportUser && (
        <div ref={menuRef} className="relative">
          <button
            onClick={() => setMenuOpen((prev) => !prev)}
            className={cn(
              "rounded-xl p-2 text-gray-400 transition-all duration-200",
              "hover:bg-gray-100 hover:text-gray-600",
              "dark:hover:bg-gray-800 dark:hover:text-gray-300",
            )}
            aria-label="More options"
          >
            <MoreVertical className="h-4 w-4" strokeWidth={1.5} />
          </button>
          {menuOpen && (
            <div
              className={cn(
                "absolute right-0 top-full z-10 mt-1 w-48 overflow-hidden rounded-lg",
                "border border-gray-100 bg-white shadow-lg",
                "dark:border-gray-700 dark:bg-gray-800",
                "animate-in fade-in slide-in-from-top-1 duration-150",
              )}
            >
              <button
                onClick={() => {
                  setMenuOpen(false)
                  onReportUser()
                }}
                className={cn(
                  "flex w-full items-center gap-2 px-3 py-2.5 text-sm text-red-600",
                  "transition-colors hover:bg-red-50",
                  "dark:text-red-400 dark:hover:bg-red-500/10",
                )}
              >
                <Flag className="h-4 w-4" strokeWidth={1.5} />
                {tReport("reportUser")}
              </button>
            </div>
          )}
        </div>
      )}

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
