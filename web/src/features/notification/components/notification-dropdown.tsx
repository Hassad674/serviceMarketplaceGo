"use client"

import { useTranslations } from "next-intl"
import { Check, BellOff } from "lucide-react"
import { Link } from "@i18n/navigation"
import { useNotifications } from "../hooks/use-notifications"
import { useMarkAllAsRead } from "../hooks/use-notification-actions"
import { NotificationItem } from "./notification-item"

import { Button } from "@/shared/components/ui/button"
interface NotificationDropdownProps {
  onClose: () => void
}

export function NotificationDropdown({ onClose }: NotificationDropdownProps) {
  const t = useTranslations("notifications")
  const { data, isLoading } = useNotifications()
  const markAllAsRead = useMarkAllAsRead()

  const notifications = data?.pages.flatMap((page) => page.data) ?? []
  const displayed = notifications.slice(0, 20)

  return (
    <div className="absolute right-0 top-full z-50 mt-2 w-96 overflow-hidden rounded-xl border border-slate-200 bg-white shadow-lg dark:border-slate-700 dark:bg-slate-800">
      {/* Header */}
      <div className="flex items-center justify-between border-b border-slate-200 px-4 py-3 dark:border-slate-700">
        <h3 className="text-sm font-semibold text-slate-900 dark:text-slate-100">
          {t("title")}
        </h3>
        <Button variant="ghost" size="auto"
          onClick={() => markAllAsRead.mutate()}
          className="flex items-center gap-1 rounded-md px-2 py-1 text-xs text-slate-500 transition-colors hover:bg-slate-100 hover:text-slate-700 dark:hover:bg-slate-700 dark:hover:text-slate-300"
          disabled={markAllAsRead.isPending}
        >
          <Check className="h-3 w-3" />
          {t("markAllAsRead")}
        </Button>
      </div>

      {/* List */}
      <div className="max-h-96 overflow-y-auto">
        {isLoading ? (
          <div className="space-y-1 p-2">
            {Array.from({ length: 5 }).map((_, i) => (
              <div key={i} className="h-16 animate-pulse rounded-lg bg-slate-100 dark:bg-slate-700" />
            ))}
          </div>
        ) : displayed.length === 0 ? (
          <div className="flex flex-col items-center gap-2 px-4 py-8 text-center">
            <BellOff className="h-8 w-8 text-slate-300 dark:text-slate-600" />
            <p className="text-sm text-slate-500 dark:text-slate-400">{t("empty")}</p>
          </div>
        ) : (
          <div className="divide-y divide-slate-100 dark:divide-slate-700">
            {displayed.map((n) => (
              <NotificationItem key={n.id} notification={n} onClose={onClose} />
            ))}
          </div>
        )}
      </div>

      {/* Footer */}
      <div className="border-t border-slate-200 dark:border-slate-700">
        <Link
          href="/notifications"
          onClick={onClose}
          className="block py-2.5 text-center text-xs font-medium text-rose-500 transition-colors hover:bg-slate-50 dark:hover:bg-slate-700"
        >
          {t("seeAll")}
        </Link>
      </div>
    </div>
  )
}
