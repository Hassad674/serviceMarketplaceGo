"use client"

import { useRef, useEffect } from "react"
import { useTranslations } from "next-intl"
import { BellOff, Loader2 } from "lucide-react"
import { useNotifications } from "../hooks/use-notifications"
import { useMarkAllAsRead } from "../hooks/use-notification-actions"
import { NotificationItem } from "./notification-item"

import { Button } from "@/shared/components/ui/button"
export function NotificationList() {
  const t = useTranslations("notifications")
  const { data, isLoading, fetchNextPage, hasNextPage, isFetchingNextPage } = useNotifications()
  const markAllAsRead = useMarkAllAsRead()
  const loadMoreRef = useRef<HTMLDivElement>(null)

  const notifications = data?.pages.flatMap((page) => page.data) ?? []

  useEffect(() => {
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting && hasNextPage && !isFetchingNextPage) {
          fetchNextPage()
        }
      },
      { threshold: 0.1 },
    )
    const el = loadMoreRef.current
    if (el) observer.observe(el)
    return () => { if (el) observer.unobserve(el) }
  }, [hasNextPage, isFetchingNextPage, fetchNextPage])

  return (
    <div className="mx-auto max-w-2xl">
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-bold text-slate-900 dark:text-slate-100">{t("title")}</h1>
        {notifications.length > 0 && (
          <Button variant="ghost" size="auto"
            onClick={() => markAllAsRead.mutate()}
            disabled={markAllAsRead.isPending}
            className="rounded-lg px-3 py-1.5 text-sm font-medium text-rose-500 transition-colors hover:bg-rose-50 dark:hover:bg-rose-900/20"
          >
            {t("markAllAsRead")}
          </Button>
        )}
      </div>

      {isLoading ? (
        <div className="space-y-2">
          {Array.from({ length: 8 }).map((_, i) => (
            <div key={i} className="h-20 animate-pulse rounded-xl bg-slate-100 dark:bg-slate-800" />
          ))}
        </div>
      ) : notifications.length === 0 ? (
        <div className="flex flex-col items-center gap-3 py-16 text-center">
          <BellOff className="h-12 w-12 text-slate-300 dark:text-slate-600" />
          <p className="text-lg font-medium text-slate-500 dark:text-slate-400">{t("empty")}</p>
          <p className="text-sm text-slate-400 dark:text-slate-500">{t("emptyDescription")}</p>
        </div>
      ) : (
        <div className="divide-y divide-slate-100 rounded-xl border border-slate-200 bg-white dark:divide-slate-700 dark:border-slate-700 dark:bg-slate-800">
          {notifications.map((n) => (
            <NotificationItem key={n.id} notification={n} />
          ))}
        </div>
      )}

      <div ref={loadMoreRef} className="flex justify-center py-4">
        {isFetchingNextPage && <Loader2 className="h-5 w-5 animate-spin text-slate-400" />}
      </div>
    </div>
  )
}
