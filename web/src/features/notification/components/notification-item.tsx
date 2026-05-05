"use client"

// Soleil v2 — single notification row.
//
// Anatomy mirrors `AppNotifications` in
// `design/assets/sources/phase1/soleil-app-lot4.jsx` (lines 123-153):
//   - 36px rounded-square icon chip (radius 11px) tinted with one of three
//     accents (corail / sapin / mute) depending on the notification type
//   - Fraunces-leaning sans-serif title (text-[13px], weight 700 if unread,
//     600 otherwise) — the JSX uses Inter Tight for the title; we follow.
//   - tabac body with relaxed line-height, truncated to 2 lines
//   - mono pill on the right with the relative timestamp (Geist Mono)
//   - corail unread dot (7px) on the trailing edge for unread items
//
// Background: ivoire-tinted `#fffaf3` for unread (matches JSX), surface for
// read. Hover raises a soft tint. The whole row is a `<button>` so click
// fires `markAsRead`.

import { useTranslations } from "next-intl"
import {
  FileText,
  Star,
  MessageCircle,
  Bell,
  Wallet,
  CheckCircle2,
  XCircle,
  RefreshCw,
  Briefcase,
  Sparkles,
} from "lucide-react"
import { useMarkAsRead } from "../hooks/use-notification-actions"
import type { Notification, NotificationType } from "../types"

type Accent = "accent" | "success" | "mute"

const iconMap: Record<NotificationType, React.ElementType> = {
  proposal_received: Briefcase,
  proposal_accepted: CheckCircle2,
  proposal_declined: XCircle,
  proposal_modified: RefreshCw,
  proposal_paid: Wallet,
  completion_requested: CheckCircle2,
  proposal_completed: CheckCircle2,
  review_received: Star,
  new_message: MessageCircle,
  system_announcement: Sparkles,
}

const accentMap: Record<NotificationType, Accent> = {
  proposal_received: "success",
  proposal_accepted: "success",
  proposal_declined: "accent",
  proposal_modified: "accent",
  proposal_paid: "success",
  completion_requested: "success",
  proposal_completed: "success",
  review_received: "accent",
  new_message: "accent",
  system_announcement: "mute",
}

// Tinted icon-chip palette — ivoire-soft for the mute branch keeps decorative
// system rows from competing visually with actionable corail/sapin notifs.
const accentChipClass: Record<Accent, string> = {
  accent: "bg-[var(--primary-soft)] text-[var(--primary)]",
  success: "bg-[var(--success-soft)] text-[var(--success)]",
  mute: "bg-[var(--background)] text-[var(--muted-foreground)]",
}

interface NotificationItemProps {
  notification: Notification
  onClose?: () => void
}

export function NotificationItem({ notification, onClose }: NotificationItemProps) {
  const t = useTranslations("notifications")
  const markAsRead = useMarkAsRead()
  const Icon = iconMap[notification.type] ?? Bell
  const accent = accentMap[notification.type] ?? "mute"
  const chipClass = accentChipClass[accent]
  const isUnread = !notification.read_at

  function handleClick() {
    if (isUnread) {
      markAsRead.mutate(notification.id)
    }
    onClose?.()
  }

  return (
    <button
      type="button"
      onClick={handleClick}
      className={`group flex w-full items-start gap-3 px-4 py-3 text-left transition-colors duration-150 hover:bg-[var(--background)] ${
        isUnread ? "bg-[#fffaf3]" : "bg-[var(--surface)]"
      }`}
    >
      <span
        aria-hidden="true"
        className={`flex h-9 w-9 shrink-0 items-center justify-center rounded-[11px] ${chipClass}`}
      >
        <Icon className="h-[15px] w-[15px]" strokeWidth={1.8} />
      </span>
      <span className="min-w-0 flex-1">
        <span className="flex items-baseline justify-between gap-2">
          <span
            className={`truncate text-[13px] leading-snug text-[var(--foreground)] ${
              isUnread ? "font-bold" : "font-semibold"
            }`}
          >
            {notification.title}
          </span>
          <span className="shrink-0 font-mono text-[10.5px] tracking-[0.04em] text-[var(--muted-foreground)]">
            {formatRelativeFr(notification.created_at, t)}
          </span>
        </span>
        {notification.body && (
          <span className="mt-0.5 line-clamp-2 text-[11.5px] leading-relaxed text-[var(--muted-foreground)]">
            {notification.body}
          </span>
        )}
      </span>
      {isUnread && (
        <span
          aria-hidden="true"
          className="mt-[7px] h-[7px] w-[7px] shrink-0 rounded-full bg-[var(--primary)]"
        />
      )}
    </button>
  )
}

// Relative time formatter — French conventions per design/DESIGN_SYSTEM.md §9
// ("à l'instant", "il y a 14 min", "il y a 1 h", "il y a 2 j").
function formatRelativeFr(
  iso: string,
  t: ReturnType<typeof useTranslations>,
): string {
  const seconds = Math.floor((Date.now() - new Date(iso).getTime()) / 1000)
  if (seconds < 60) return t("timeJustNow")
  const minutes = Math.floor(seconds / 60)
  if (minutes < 60) return t("timeMinutes", { n: minutes })
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return t("timeHours", { n: hours })
  const days = Math.floor(hours / 24)
  return t("timeDays", { n: days })
}
