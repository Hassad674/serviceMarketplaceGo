"use client"

import { useState, useMemo } from "react"
import Image from "next/image"
import { Search, Minus, Pencil, MessageCircle } from "lucide-react"
import { useTranslations } from "next-intl"
import { Link } from "@i18n/navigation"
import { cn } from "@/shared/lib/utils"
import { Portrait } from "@/shared/components/ui/portrait"
import type { Conversation } from "@/features/messaging/types"

import { Button } from "@/shared/components/ui/button"
import { Input } from "@/shared/components/ui/input"

type TypingState = Record<string, { userId: string }>

type WidgetFilterKey = "all" | "unread" | "archived"

interface ChatWidgetConversationListProps {
  conversations: Conversation[]
  isLoading: boolean
  typingUsers: TypingState
  onSelect: (id: string) => void
  onClose: () => void
}

// ChatWidgetConversationList — Soleil v2 widget list view.
//
// Layout (per design/diffs/MESSAGERIE-WIDGET/target/widget-pdf.pdf
// pages 4 & 7):
//   header (Messages + count + new + minimize)
//   search input
//   tabs (Tous / Non lus / Archivés)
//   scrollable list of compact rows (Portrait + name + preview + time + unread dot)
//   footer link "Voir tous les messages — N conversations"
//
// All hooks and data come unchanged from the parent panel — this file
// only owns presentation.
function formatRelativeTime(dateStr: string, locale = "fr"): string {
  const date = new Date(dateStr)
  const now = new Date()
  const sameDay =
    date.toDateString() === now.toDateString()
  if (sameDay) {
    return date.toLocaleTimeString(locale === "en" ? undefined : "fr-FR", {
      hour: "2-digit",
      minute: "2-digit",
    })
  }
  const yesterday = new Date(now)
  yesterday.setDate(yesterday.getDate() - 1)
  if (date.toDateString() === yesterday.toDateString()) {
    return locale === "en" ? "Yesterday" : "Hier"
  }
  const diffDays = Math.floor((now.getTime() - date.getTime()) / 86_400_000)
  if (diffDays < 7) {
    const day = date.toLocaleDateString(locale === "en" ? undefined : "fr-FR", {
      weekday: "short",
    })
    // Capitalize + trailing dot for FR ("Lun.")
    return locale === "en" ? day : day.charAt(0).toUpperCase() + day.slice(1).replace(/\.$/, "") + "."
  }
  return date.toLocaleDateString(locale === "en" ? undefined : "fr-FR", {
    day: "numeric",
    month: "short",
  })
}

// Stable portrait id derived from the conversation id so the avatar
// stays consistent across re-renders without keeping a shared lookup.
function portraitIdFor(seed: string): number {
  let h = 0
  for (let i = 0; i < seed.length; i++) {
    h = (h * 31 + seed.charCodeAt(i)) >>> 0
  }
  return h % 6
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
  const [filter, setFilter] = useState<WidgetFilterKey>("all")

  const totalUnread = useMemo(
    () =>
      conversations.reduce((acc, c) => acc + (c.unread_count > 0 ? 1 : 0), 0),
    [conversations],
  )

  const filtered = useMemo(() => {
    return conversations.filter((c) => {
      if (
        searchQuery &&
        !c.other_org_name.toLowerCase().includes(searchQuery.toLowerCase())
      ) {
        return false
      }
      if (filter === "unread") return c.unread_count > 0
      // Archived view is not yet supported by the API — keep the tab in
      // sync with the design (so the widget UI matches the maquette) but
      // surface a neutral empty state when picked.
      if (filter === "archived") return false
      return true
    })
  }, [conversations, searchQuery, filter])

  return (
    <div className="flex h-full flex-col">
      {/* Header */}
      <div className="flex items-center gap-2 px-5 pb-3 pt-4">
        <h2 className="font-serif text-[20px] font-medium leading-none tracking-tight text-foreground">
          {t("messagingWidget_title")}
        </h2>
        {totalUnread > 0 && (
          <span className="inline-flex h-5 min-w-[20px] items-center justify-center rounded-full bg-primary px-1.5 text-[11px] font-semibold leading-none text-primary-foreground">
            {totalUnread}
          </span>
        )}
        <div className="flex-1" />
        <Button
          variant="ghost"
          size="auto"
          className="rounded-full p-1.5 text-muted-foreground transition-colors hover:bg-primary-soft hover:text-primary-deep"
          aria-label={t("messagingWidget_compose")}
        >
          <Pencil className="h-4 w-4" strokeWidth={1.6} />
        </Button>
        <Button
          variant="ghost"
          size="auto"
          onClick={onClose}
          className="rounded-full p-1.5 text-muted-foreground transition-colors hover:bg-border hover:text-foreground"
          aria-label={t("messagingWidget_close")}
        >
          <Minus className="h-4 w-4" strokeWidth={1.6} />
        </Button>
      </div>

      {/* Search */}
      <div className="px-5 pb-3">
        <div className="relative">
          <Search
            className="pointer-events-none absolute left-3.5 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground"
            strokeWidth={1.6}
          />
          <Input
            type="text"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder={t("messagingWidget_search")}
            aria-label={t("messagingWidget_search")}
            className={cn(
              "h-10 w-full rounded-xl border border-border bg-background pl-9 pr-3 text-sm text-foreground",
              "placeholder:text-muted-foreground transition-all duration-200",
              "focus:border-primary focus:bg-card focus:outline-none focus:ring-2 focus:ring-primary/20",
            )}
          />
        </div>
      </div>

      {/* Tabs */}
      <div className="flex items-center gap-5 border-b border-border px-5">
        <FilterTab
          label={t("messagingWidget_filterAll")}
          count={conversations.length}
          active={filter === "all"}
          onClick={() => setFilter("all")}
        />
        <FilterTab
          label={t("messagingWidget_filterUnread")}
          count={totalUnread}
          active={filter === "unread"}
          onClick={() => setFilter("unread")}
        />
        <FilterTab
          label={t("messagingWidget_filterArchived")}
          active={filter === "archived"}
          onClick={() => setFilter("archived")}
        />
      </div>

      {/* List */}
      <div className="flex-1 overflow-y-auto" role="listbox" aria-label={t("messagingWidget_title")}>
        {isLoading ? (
          <ConversationListSkeleton />
        ) : filtered.length === 0 ? (
          <div className="flex h-full flex-col items-center justify-center gap-2 px-5 py-10 text-center">
            <span className="flex h-12 w-12 items-center justify-center rounded-2xl bg-primary-soft text-primary-deep">
              <MessageCircle className="h-5 w-5" strokeWidth={1.5} />
            </span>
            <p className="font-serif text-[15px] font-medium text-foreground">
              {t("messagingWidget_emptyTitle")}
            </p>
            <p className="text-xs text-muted-foreground">
              {t("messagingWidget_emptyBody")}
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

      {/* Footer */}
      <div className="flex items-center justify-between border-t border-border bg-card px-5 py-3">
        <Link
          href="/messages"
          className="text-sm font-semibold text-primary transition-colors hover:text-primary-deep"
        >
          {t("messagingWidget_viewAll")}
        </Link>
        <span className="font-mono text-[11px] text-muted-foreground">
          {t("messagingWidget_summary", { count: conversations.length })}
        </span>
      </div>
    </div>
  )
}

interface FilterTabProps {
  label: string
  count?: number
  active: boolean
  onClick: () => void
}

function FilterTab({ label, count, active, onClick }: FilterTabProps) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        "relative flex items-center gap-1.5 py-3 text-sm font-medium transition-colors",
        active ? "text-foreground" : "text-muted-foreground hover:text-foreground",
      )}
    >
      <span>{label}</span>
      {typeof count === "number" && count > 0 && (
        <span
          className={cn(
            "inline-flex h-[18px] min-w-[18px] items-center justify-center rounded-full px-1 text-[10px] font-semibold leading-none",
            active
              ? "bg-primary text-primary-foreground"
              : "bg-border text-muted-foreground",
          )}
        >
          {count}
        </span>
      )}
      {active && (
        <span className="absolute -bottom-px left-0 right-0 h-[2px] rounded-full bg-primary" />
      )}
    </button>
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

  return (
    <Button
      variant="ghost"
      size="auto"
      onClick={() => onSelect(conversation.id)}
      role="option"
      aria-selected={false}
      className={cn(
        "flex w-full items-center gap-3 px-5 py-3 text-left transition-colors duration-150",
        "border-l-[3px] border-l-transparent",
        "hover:bg-primary-soft/40",
      )}
    >
      {/* Avatar */}
      <div className="relative shrink-0">
        {conversation.other_photo_url ? (
          <Image
            src={conversation.other_photo_url}
            alt={conversation.other_org_name}
            width={40}
            height={40}
            className="h-10 w-10 rounded-full object-cover"
            unoptimized
          />
        ) : (
          <Portrait
            id={portraitIdFor(conversation.id)}
            size={40}
            alt={conversation.other_org_name}
          />
        )}
        {conversation.online && (
          <span
            className="absolute bottom-0 right-0 h-2.5 w-2.5 rounded-full border-2 border-card bg-emerald-500"
            aria-label={t("online")}
          />
        )}
      </div>

      {/* Content */}
      <div className="min-w-0 flex-1">
        <div className="flex items-baseline justify-between gap-2">
          <p className="truncate text-[14px] font-semibold text-foreground">
            {conversation.other_org_name}
          </p>
          {conversation.last_message_at && (
            <span className="shrink-0 font-mono text-[11px] text-muted-foreground">
              {formatRelativeTime(conversation.last_message_at)}
            </span>
          )}
        </div>
        <div className="mt-0.5 flex items-center justify-between gap-2">
          {isTyping ? (
            <p className="truncate text-[12px] italic text-primary">
              {t("typingShort")}
            </p>
          ) : (
            <p
              className={cn(
                "truncate text-[12px]",
                conversation.unread_count > 0
                  ? "font-medium text-foreground"
                  : "text-muted-foreground",
              )}
            >
              {conversation.last_message ?? t("noMessages")}
            </p>
          )}
          {conversation.unread_count > 0 && (
            <span className="ml-1 inline-flex h-[18px] min-w-[18px] shrink-0 items-center justify-center rounded-full bg-primary px-1 text-[10px] font-semibold leading-none text-primary-foreground">
              {conversation.unread_count}
            </span>
          )}
        </div>
      </div>
    </Button>
  )
}

function ConversationListSkeleton() {
  return (
    <div className="px-5 py-2">
      {[0, 1, 2, 3].map((i) => (
        <div key={i} className="flex items-center gap-3 py-3">
          <div className="h-10 w-10 shrink-0 animate-pulse rounded-full bg-border" />
          <div className="min-w-0 flex-1 space-y-1.5">
            <div className="h-3 w-28 animate-pulse rounded bg-border" />
            <div className="h-2.5 w-44 animate-pulse rounded bg-border" />
          </div>
        </div>
      ))}
    </div>
  )
}
