"use client"

import { Search } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import type { Conversation } from "../types"

const ROLE_FILTERS = [
  { key: "all", labelKey: "allRoles" },
  { key: "agency", labelKey: "agency" },
  { key: "provider", labelKey: "freelancer" },
  { key: "enterprise", labelKey: "enterprise" },
]

const ROLE_PILL_STYLES: Record<string, string> = {
  all: "bg-gray-100 text-gray-700 dark:bg-gray-700 dark:text-gray-300",
  agency: "bg-blue-50 text-blue-700 dark:bg-blue-500/20 dark:text-blue-400",
  provider: "bg-rose-50 text-rose-700 dark:bg-rose-500/20 dark:text-rose-400",
  enterprise: "bg-purple-50 text-purple-700 dark:bg-purple-500/20 dark:text-purple-400",
}

const ROLE_PILL_ACTIVE: Record<string, string> = {
  all: "bg-gray-900 text-white dark:bg-gray-100 dark:text-gray-900",
  agency: "bg-blue-600 text-white dark:bg-blue-500 dark:text-white",
  provider: "bg-rose-600 text-white dark:bg-rose-500 dark:text-white",
  enterprise: "bg-purple-600 text-white dark:bg-purple-500 dark:text-white",
}

const ROLE_BORDER_COLORS: Record<string, string> = {
  agency: "border-l-blue-500",
  provider: "border-l-rose-500",
  enterprise: "border-l-purple-500",
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

type TypingEntry = { userId: string }

type TypingState = Record<string, TypingEntry>

interface ConversationListProps {
  conversations: Conversation[]
  activeId: string | null
  roleFilter: string
  searchQuery: string
  typingUsers: TypingState
  onSelect: (id: string) => void
  onRoleFilterChange: (role: string) => void
  onSearchChange: (query: string) => void
}

export function ConversationList({
  conversations,
  activeId,
  roleFilter,
  searchQuery,
  typingUsers,
  onSelect,
  onRoleFilterChange,
  onSearchChange,
}: ConversationListProps) {
  const t = useTranslations("messaging")

  const filtered = conversations.filter((c) => {
    const matchesRole = roleFilter === "all" || c.other_user_role === roleFilter
    const matchesSearch =
      !searchQuery ||
      c.other_user_name.toLowerCase().includes(searchQuery.toLowerCase())
    return matchesRole && matchesSearch
  })

  return (
    <div className="flex h-full flex-col">
      {/* Title */}
      <div className="px-5 pt-5 pb-3">
        <h1 className="text-xl font-bold text-gray-900 dark:text-white">
          {t("title")}
        </h1>
      </div>

      {/* Role filter tabs */}
      <div className="flex gap-1.5 px-5 pb-3">
        {ROLE_FILTERS.map((filter) => (
          <button
            key={filter.key}
            onClick={() => onRoleFilterChange(filter.key)}
            className={cn(
              "rounded-full px-3 py-1.5 text-xs font-medium transition-all duration-200",
              roleFilter === filter.key
                ? ROLE_PILL_ACTIVE[filter.key]
                : cn(ROLE_PILL_STYLES[filter.key], "hover:opacity-80"),
            )}
          >
            {t(filter.labelKey)}
          </button>
        ))}
      </div>

      {/* Search */}
      <div className="px-5 pb-3">
        <div className="relative">
          <Search
            className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400 dark:text-gray-500"
            strokeWidth={1.5}
          />
          <input
            type="text"
            value={searchQuery}
            onChange={(e) => onSearchChange(e.target.value)}
            placeholder={t("searchPlaceholder")}
            className={cn(
              "h-9 w-full rounded-lg bg-gray-100/80 pl-9 pr-4 text-sm text-gray-900",
              "placeholder:text-gray-400 transition-all duration-200",
              "focus:bg-white focus:shadow-sm focus:ring-2 focus:ring-rose-500/20 focus:outline-none",
              "dark:bg-gray-800/80 dark:text-gray-100 dark:placeholder:text-gray-500",
              "dark:focus:bg-gray-800 dark:focus:ring-rose-400/20",
            )}
          />
        </div>
      </div>

      {/* Conversation items */}
      <div className="flex-1 overflow-y-auto">
        {filtered.length === 0 ? (
          <div className="px-5 py-8 text-center">
            <p className="text-sm text-gray-400 dark:text-gray-500">
              {t("noConversations")}
            </p>
          </div>
        ) : (
          filtered.map((conversation) => (
            <ConversationItem
              key={conversation.id}
              conversation={conversation}
              isActive={conversation.id === activeId}
              isTyping={!!typingUsers[conversation.id]}
              onSelect={onSelect}
            />
          ))
        )}
      </div>
    </div>
  )
}

interface ConversationItemProps {
  conversation: Conversation
  isActive: boolean
  isTyping: boolean
  onSelect: (id: string) => void
}

function ConversationItem({
  conversation,
  isActive,
  isTyping,
  onSelect,
}: ConversationItemProps) {
  const t = useTranslations("messaging")
  const initials = conversation.other_user_name
    .split(" ")
    .map((w) => w.charAt(0))
    .join("")
    .slice(0, 2)
    .toUpperCase()

  const borderColor =
    ROLE_BORDER_COLORS[conversation.other_user_role] ?? "border-l-gray-300"

  return (
    <button
      onClick={() => onSelect(conversation.id)}
      className={cn(
        "flex w-full items-center gap-3 border-l-[3px] px-5 py-3 text-left transition-all duration-200",
        isActive
          ? cn("bg-gray-50 dark:bg-gray-800/50", borderColor)
          : "border-l-transparent hover:bg-gray-50/50 dark:hover:bg-gray-800/30",
      )}
    >
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

      {/* Content */}
      <div className="min-w-0 flex-1">
        <div className="flex items-center justify-between">
          <p className="truncate text-sm font-semibold text-gray-900 dark:text-white">
            {conversation.other_user_name}
          </p>
          {conversation.last_message_at && (
            <span className="ml-2 shrink-0 text-xs text-gray-400 dark:text-gray-500">
              {formatRelativeTime(conversation.last_message_at)}
            </span>
          )}
        </div>
        <div className="mt-0.5 flex items-center justify-between">
          {isTyping ? (
            <p className="truncate text-xs italic text-rose-500 dark:text-rose-400">
              {t("typingShort")}
            </p>
          ) : (
            <p className="truncate text-xs text-gray-500 dark:text-gray-400">
              {conversation.last_message ?? t("noMessages")}
            </p>
          )}
          {conversation.unread_count > 0 && (
            <span className="ml-2 flex h-5 min-w-5 shrink-0 items-center justify-center rounded-full bg-rose-500 px-1.5 text-[10px] font-bold text-white">
              {conversation.unread_count}
            </span>
          )}
        </div>
      </div>
    </button>
  )
}
