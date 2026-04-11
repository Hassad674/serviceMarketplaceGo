"use client"

import { useState, useMemo } from "react"
import Image from "next/image"
import { Search, ChevronDown, MessageSquare } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import type { Conversation } from "@/features/messaging/types"

type TypingState = Record<string, { userId: string }>

interface ChatWidgetConversationListProps {
  conversations: Conversation[]
  isLoading: boolean
  typingUsers: TypingState
  onSelect: (id: string) => void
  onClose: () => void
}

function formatRelativeTime(dateStr: string): string {
  const date = new Date(dateStr)
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const diffMinutes = Math.floor(diffMs / 60_000)
  const diffHours = Math.floor(diffMs / 3_600_000)
  const diffDays = Math.floor(diffMs / 86_400_000)

  if (diffMinutes < 1) return "now"
  if (diffMinutes < 60) return `${diffMinutes}m`
  if (diffHours < 24) return `${diffHours}h`
  if (diffDays < 7) return `${diffDays}d`

  return date.toLocaleDateString(undefined, { month: "short", day: "numeric" })
}

export function ChatWidgetConversationList({
  conversations,
  isLoading,
  typingUsers,
  onSelect,
  onClose,
}: ChatWidgetConversationListProps) {
  const t = useTranslations("messaging")
  const [searchQuery, setSearchQuery] = useState("")

  const filtered = useMemo(
    () =>
      conversations.filter(
        (c) =>
          !searchQuery ||
          c.other_org_name.toLowerCase().includes(searchQuery.toLowerCase()),
      ),
    [conversations, searchQuery],
  )

  return (
    <div className="flex h-full flex-col">
      {/* Header — Contra style */}
      <div className="flex h-12 shrink-0 items-center gap-2.5 border-b border-gray-100 px-4 dark:border-gray-800">
        <MessageSquare
          className="h-[18px] w-[18px] text-gray-700 dark:text-gray-300"
          strokeWidth={1.5}
        />
        <h2 className="flex-1 text-sm font-semibold text-gray-900 dark:text-white">
          {t("title")}
        </h2>
        <button
          onClick={onClose}
          className="rounded-lg p-1.5 text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-600 dark:hover:bg-gray-800 dark:hover:text-gray-300"
          aria-label={t("close")}
        >
          <ChevronDown className="h-4 w-4" strokeWidth={1.5} />
        </button>
      </div>

      {/* Search */}
      <div className="shrink-0 px-3 py-2">
        <div className="relative">
          <Search
            className="absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-gray-400 dark:text-gray-500"
            strokeWidth={1.5}
          />
          <input
            type="text"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder={t("searchPlaceholder")}
            aria-label={t("searchPlaceholder")}
            className={cn(
              "h-8 w-full rounded-lg bg-gray-100/80 pl-8 pr-3 text-xs text-gray-900",
              "placeholder:text-gray-400 transition-all duration-200",
              "focus:bg-white focus:shadow-sm focus:ring-2 focus:ring-rose-500/20 focus:outline-none",
              "dark:bg-gray-800/80 dark:text-gray-100 dark:placeholder:text-gray-500",
              "dark:focus:bg-gray-800 dark:focus:ring-rose-400/20",
            )}
          />
        </div>
      </div>

      {/* Conversation items */}
      <div className="flex-1 overflow-y-auto" role="listbox" aria-label={t("title")}>
        {isLoading ? (
          <ConversationListSkeleton />
        ) : filtered.length === 0 ? (
          <div className="px-4 py-6 text-center">
            <p className="text-xs text-gray-400 dark:text-gray-500">
              {t("noConversations")}
            </p>
          </div>
        ) : (
          filtered.map((conversation) => (
            <CompactConversationItem
              key={conversation.id}
              conversation={conversation}
              isTyping={!!typingUsers[conversation.id]}
              onSelect={onSelect}
            />
          ))
        )}
      </div>

    </div>
  )
}

interface CompactConversationItemProps {
  conversation: Conversation
  isTyping: boolean
  onSelect: (id: string) => void
}

function CompactConversationItem({
  conversation,
  isTyping,
  onSelect,
}: CompactConversationItemProps) {
  const t = useTranslations("messaging")
  const initials = conversation.other_org_name
    .split(" ")
    .map((w: string) => w.charAt(0))
    .join("")
    .slice(0, 2)
    .toUpperCase()

  return (
    <button
      onClick={() => onSelect(conversation.id)}
      role="option"
      aria-selected={false}
      className={cn(
        "flex w-full items-center gap-2.5 px-3 py-2.5 text-left transition-all duration-200",
        "hover:bg-gray-50/80 dark:hover:bg-gray-800/30",
      )}
    >
      {/* Avatar */}
      <div className="relative shrink-0">
        {conversation.other_photo_url ? (
          <Image
            src={conversation.other_photo_url}
            alt={conversation.other_org_name}
            width={36}
            height={36}
            className="h-9 w-9 rounded-full object-cover"
            unoptimized
          />
        ) : (
          <div className="flex h-9 w-9 items-center justify-center rounded-full bg-gradient-to-br from-rose-500 to-purple-600 text-xs font-semibold text-white">
            {initials}
          </div>
        )}
        {conversation.online && (
          <span className="absolute bottom-0 right-0 h-2.5 w-2.5 rounded-full border-2 border-white bg-emerald-500 dark:border-gray-900" />
        )}
      </div>

      {/* Content */}
      <div className="min-w-0 flex-1">
        <div className="flex items-center justify-between">
          <p className="truncate text-xs font-semibold text-gray-900 dark:text-white">
            {conversation.other_org_name}
          </p>
          {conversation.last_message_at && (
            <span className="ml-2 shrink-0 text-[10px] text-gray-400 dark:text-gray-500">
              {formatRelativeTime(conversation.last_message_at)}
            </span>
          )}
        </div>
        <div className="mt-0.5 flex items-center justify-between">
          {isTyping ? (
            <p className="truncate text-[11px] italic text-rose-500 dark:text-rose-400">
              {t("typingShort")}
            </p>
          ) : (
            <p className="truncate text-[11px] text-gray-500 dark:text-gray-400">
              {conversation.last_message ?? t("noMessages")}
            </p>
          )}
          {conversation.unread_count > 0 && (
            <span className="ml-2 flex h-4 min-w-4 shrink-0 items-center justify-center rounded-full bg-rose-500 px-1 text-[9px] font-bold text-white">
              {conversation.unread_count}
            </span>
          )}
        </div>
      </div>
    </button>
  )
}

function ConversationListSkeleton() {
  return (
    <div className="space-y-1 px-3 py-2">
      {[1, 2, 3, 4].map((i) => (
        <div key={i} className="flex items-center gap-2.5 py-2.5">
          <div className="h-9 w-9 shrink-0 animate-pulse rounded-full bg-gray-200 dark:bg-gray-700" />
          <div className="min-w-0 flex-1 space-y-1.5">
            <div className="h-3 w-24 animate-pulse rounded bg-gray-200 dark:bg-gray-700" />
            <div className="h-2.5 w-36 animate-pulse rounded bg-gray-200 dark:bg-gray-700" />
          </div>
        </div>
      ))}
    </div>
  )
}
