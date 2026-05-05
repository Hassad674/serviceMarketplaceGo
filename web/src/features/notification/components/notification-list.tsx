"use client"

// Soleil v2 — full-page notification list (W-24).
//
// Layout adapts the mobile `AppNotifications` source
// (`design/assets/sources/phase1/soleil-app-lot4.jsx`, lines 72-121) to a
// desktop column. Each anchor of the screen:
//   - Editorial Fraunces title with italic corail accent ("Notifications
//     récentes") + Fraunces italic subtitle ("5 non lues · tout marquer lu"
//     pattern from the JSX source line 78). The subtitle doubles as the
//     "mark all as read" affordance — clicking it fires the existing
//     `useMarkAllAsRead` mutation.
//   - Notifications grouped chronologically (Aujourd'hui / Hier /
//     Cette semaine / Plus ancien) under mono uppercase eyebrows
//     (text-subtle-foreground, 0.06em tracking, weight 700).
//   - Each group is a rounded ivoire card (rounded-2xl, sable border,
//     calm shadow-card) with rows separated by 1px sable dividers.
//   - Empty state: corail-soft circular icon chip + Fraunces title +
//     italic Fraunces subtitle in tabac (matches the calm Soleil empty
//     pattern used across wallet / invoices).
//
// Filter pills: backend doesn't expose a typed filter on the listing
// endpoint (only `cursor` + `limit`), so we SKIP filter pills and FLAG
// in the report. The grouping itself acts as the visual segmenter.

import { useRef, useEffect, useMemo } from "react"
import { useTranslations } from "next-intl"
import { Bell, Loader2 } from "lucide-react"
import { useNotifications } from "../hooks/use-notifications"
import { useMarkAllAsRead } from "../hooks/use-notification-actions"
import { NotificationItem } from "./notification-item"
import type { Notification } from "../types"

type GroupKey = "today" | "yesterday" | "thisWeek" | "earlier"

interface NotificationGroup {
  key: GroupKey
  label: string
  items: Notification[]
}

export function NotificationList() {
  const t = useTranslations("notifications")
  const { data, isLoading, fetchNextPage, hasNextPage, isFetchingNextPage } =
    useNotifications()
  const markAllAsRead = useMarkAllAsRead()
  const loadMoreRef = useRef<HTMLDivElement>(null)

  const notifications = useMemo(
    () => data?.pages.flatMap((page) => page.data) ?? [],
    [data],
  )

  // Unread count derived from the loaded pages — keeps the UI a pure
  // function of the data already in cache, avoids adding a second hook
  // dependency for the same number.
  const unreadCount = useMemo(
    () => notifications.filter((n) => n.read_at === null).length,
    [notifications],
  )

  const groups = useMemo(() => {
    const labels: Record<GroupKey, string> = {
      today: t("groupToday"),
      yesterday: t("groupYesterday"),
      thisWeek: t("groupThisWeek"),
      earlier: t("groupEarlier"),
    }
    return groupByRelativeDay(notifications, labels)
  }, [notifications, t])

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
    return () => {
      if (el) observer.unobserve(el)
    }
  }, [hasNextPage, isFetchingNextPage, fetchNextPage])

  const hasNotifications = notifications.length > 0
  const subtitleIsAction = unreadCount > 0
  const subtitle = (() => {
    if (!hasNotifications) return t("subtitleNone")
    if (unreadCount === 0) return t("subtitleAllRead")
    return t("subtitleUnread", { count: unreadCount })
  })()

  return (
    <div className="mx-auto w-full max-w-2xl">
      {/* Editorial header — Fraunces title with italic corail accent.
          The italic word ("récentes") is the Soleil signature treatment. */}
      <header className="mb-6 px-1">
        <h1 className="font-serif text-[28px] font-medium leading-tight tracking-[-0.02em] text-[var(--foreground)] sm:text-[32px]">
          {t("title")}{" "}
          <span className="italic text-[var(--primary)]">
            {t("titleAccent")}
          </span>
        </h1>
        <div className="mt-1 text-[13px]">
          {hasNotifications && subtitleIsAction ? (
            <button
              type="button"
              onClick={() => markAllAsRead.mutate()}
              disabled={markAllAsRead.isPending}
              className="font-serif italic text-[var(--muted-foreground)] underline-offset-4 transition-colors hover:text-[var(--primary)] hover:underline disabled:opacity-60"
            >
              {subtitle}
            </button>
          ) : (
            <p className="font-serif italic text-[var(--muted-foreground)]">
              {subtitle}
            </p>
          )}
        </div>
      </header>

      {isLoading ? (
        <NotificationsSkeleton />
      ) : !hasNotifications ? (
        <EmptyState
          title={t("empty")}
          description={t("emptyDescription")}
        />
      ) : (
        <div className="space-y-7">
          {groups.map((group) => (
            <section key={group.key} aria-label={group.label}>
              <h2 className="mx-1 mb-2.5 font-mono text-[11px] font-bold uppercase tracking-[0.06em] text-[var(--subtle-foreground)]">
                {group.label}
              </h2>
              <div className="overflow-hidden rounded-2xl border border-[var(--border)] bg-[var(--surface)] shadow-[var(--shadow-card)]">
                {group.items.map((notification, index) => (
                  <div
                    key={notification.id}
                    className={
                      index < group.items.length - 1
                        ? "border-b border-[var(--border)]"
                        : ""
                    }
                  >
                    <NotificationItem notification={notification} />
                  </div>
                ))}
              </div>
            </section>
          ))}
        </div>
      )}

      <div ref={loadMoreRef} className="flex justify-center py-6">
        {isFetchingNextPage && (
          <Loader2
            className="h-5 w-5 animate-spin text-[var(--muted-foreground)]"
            aria-hidden="true"
          />
        )}
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Sub-components
// ---------------------------------------------------------------------------

function NotificationsSkeleton() {
  return (
    <div className="space-y-7">
      {[0, 1].map((group) => (
        <div key={group}>
          <div className="mx-1 mb-2.5 h-3 w-24 animate-pulse rounded-full bg-[var(--border)]" />
          <div className="overflow-hidden rounded-2xl border border-[var(--border)] bg-[var(--surface)]">
            {[0, 1, 2].map((i) => (
              <div
                key={i}
                className={`flex items-start gap-3 px-4 py-3 ${
                  i < 2 ? "border-b border-[var(--border)]" : ""
                }`}
              >
                <div className="h-9 w-9 shrink-0 animate-pulse rounded-[11px] bg-[var(--border)]" />
                <div className="flex-1 space-y-2">
                  <div className="h-3 w-2/3 animate-pulse rounded-full bg-[var(--border)]" />
                  <div className="h-3 w-1/2 animate-pulse rounded-full bg-[var(--border)]" />
                </div>
              </div>
            ))}
          </div>
        </div>
      ))}
    </div>
  )
}

function EmptyState({
  title,
  description,
}: {
  title: string
  description: string
}) {
  return (
    <div className="overflow-hidden rounded-2xl border border-[var(--border)] bg-[var(--surface)] shadow-[var(--shadow-card)]">
      <div className="flex flex-col items-center gap-3 px-6 py-16 text-center">
        <span
          aria-hidden="true"
          className="flex h-14 w-14 items-center justify-center rounded-full bg-[var(--primary-soft)] text-[var(--primary)]"
        >
          <Bell className="h-6 w-6" strokeWidth={1.6} />
        </span>
        <p className="font-serif text-[20px] font-medium tracking-[-0.01em] text-[var(--foreground)]">
          {title}
        </p>
        <p className="max-w-xs font-serif text-[13.5px] italic leading-relaxed text-[var(--muted-foreground)]">
          {description}
        </p>
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Grouping helper — pure function, easy to unit-test if needed.
// ---------------------------------------------------------------------------

function groupByRelativeDay(
  items: Notification[],
  labels: Record<GroupKey, string>,
): NotificationGroup[] {
  const now = new Date()
  const startOfToday = new Date(
    now.getFullYear(),
    now.getMonth(),
    now.getDate(),
  ).getTime()
  const startOfYesterday = startOfToday - 86_400_000
  const startOfThisWeek = startOfToday - 6 * 86_400_000

  const buckets: Record<GroupKey, Notification[]> = {
    today: [],
    yesterday: [],
    thisWeek: [],
    earlier: [],
  }

  for (const item of items) {
    const ts = new Date(item.created_at).getTime()
    if (ts >= startOfToday) buckets.today.push(item)
    else if (ts >= startOfYesterday) buckets.yesterday.push(item)
    else if (ts >= startOfThisWeek) buckets.thisWeek.push(item)
    else buckets.earlier.push(item)
  }

  const order: GroupKey[] = ["today", "yesterday", "thisWeek", "earlier"]
  return order
    .filter((key) => buckets[key].length > 0)
    .map((key) => ({ key, label: labels[key], items: buckets[key] }))
}
