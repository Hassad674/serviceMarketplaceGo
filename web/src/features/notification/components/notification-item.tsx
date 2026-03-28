"use client"

import { FileText, Star, MessageSquare, Bell, CreditCard, CheckCircle, XCircle, RefreshCw } from "lucide-react"
import { useMarkAsRead } from "../hooks/use-notification-actions"
import type { Notification, NotificationType } from "../types"

const iconMap: Record<NotificationType, React.ElementType> = {
  proposal_received: FileText,
  proposal_accepted: CheckCircle,
  proposal_declined: XCircle,
  proposal_modified: RefreshCw,
  proposal_paid: CreditCard,
  completion_requested: CheckCircle,
  proposal_completed: CheckCircle,
  review_received: Star,
  new_message: MessageSquare,
  system_announcement: Bell,
}

const colorMap: Record<NotificationType, string> = {
  proposal_received: "bg-blue-50 text-blue-600 dark:bg-blue-900/30 dark:text-blue-400",
  proposal_accepted: "bg-green-50 text-green-600 dark:bg-green-900/30 dark:text-green-400",
  proposal_declined: "bg-red-50 text-red-600 dark:bg-red-900/30 dark:text-red-400",
  proposal_modified: "bg-amber-50 text-amber-600 dark:bg-amber-900/30 dark:text-amber-400",
  proposal_paid: "bg-emerald-50 text-emerald-600 dark:bg-emerald-900/30 dark:text-emerald-400",
  completion_requested: "bg-violet-50 text-violet-600 dark:bg-violet-900/30 dark:text-violet-400",
  proposal_completed: "bg-green-50 text-green-600 dark:bg-green-900/30 dark:text-green-400",
  review_received: "bg-amber-50 text-amber-600 dark:bg-amber-900/30 dark:text-amber-400",
  new_message: "bg-sky-50 text-sky-600 dark:bg-sky-900/30 dark:text-sky-400",
  system_announcement: "bg-slate-100 text-slate-600 dark:bg-slate-700 dark:text-slate-400",
}

function timeAgo(dateStr: string): string {
  const seconds = Math.floor((Date.now() - new Date(dateStr).getTime()) / 1000)
  if (seconds < 60) return "just now"
  const minutes = Math.floor(seconds / 60)
  if (minutes < 60) return `${minutes}m`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}h`
  const days = Math.floor(hours / 24)
  return `${days}d`
}

interface NotificationItemProps {
  notification: Notification
  onClose?: () => void
}

export function NotificationItem({ notification, onClose }: NotificationItemProps) {
  const markAsRead = useMarkAsRead()
  const Icon = iconMap[notification.type] ?? Bell
  const colors = colorMap[notification.type] ?? colorMap.system_announcement
  const isUnread = !notification.read_at

  function handleClick() {
    if (isUnread) {
      markAsRead.mutate(notification.id)
    }
    onClose?.()
  }

  return (
    <button
      onClick={handleClick}
      className={`flex w-full items-start gap-3 px-4 py-3 text-left transition-colors hover:bg-slate-50 dark:hover:bg-slate-700/50 ${
        isUnread ? "bg-rose-50/30 dark:bg-rose-900/10" : ""
      }`}
    >
      <div className={`flex h-8 w-8 shrink-0 items-center justify-center rounded-full ${colors}`}>
        <Icon className="h-4 w-4" />
      </div>
      <div className="min-w-0 flex-1">
        <p className={`truncate text-sm ${isUnread ? "font-semibold text-slate-900 dark:text-slate-100" : "text-slate-700 dark:text-slate-300"}`}>
          {notification.title}
        </p>
        {notification.body && (
          <p className="mt-0.5 truncate text-xs text-slate-500 dark:text-slate-400">
            {notification.body}
          </p>
        )}
        <p className="mt-1 text-[10px] text-slate-400 dark:text-slate-500">
          {timeAgo(notification.created_at)}
        </p>
      </div>
      {isUnread && (
        <div className="mt-2 h-2 w-2 shrink-0 rounded-full bg-rose-500" />
      )}
    </button>
  )
}
