"use client"

// Soleil v2 — topbar bell dropdown.
//
// Mirrors the full-page list anatomy at a compressed scale: ivoire surface
// with sable border + calm shadow, Fraunces title with italic corail
// "récentes" accent, mono uppercase markAllAsRead pill, and shared
// `NotificationItem` rows. Footer is a corail-tinted "see all" link.

import { useTranslations } from "next-intl"
import { Bell, Check } from "lucide-react"
import { Link } from "@i18n/navigation"
import { useNotifications } from "../hooks/use-notifications"
import { useMarkAllAsRead } from "../hooks/use-notification-actions"
import { NotificationItem } from "./notification-item"

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
    <div className="absolute right-0 top-full z-50 mt-2 w-96 overflow-hidden rounded-2xl border border-[var(--border)] bg-[var(--surface)] shadow-[var(--shadow-card)]">
      {/* Header — Fraunces title + mono uppercase mark-all action */}
      <div className="flex items-center justify-between gap-3 border-b border-[var(--border)] px-4 py-3">
        <h3 className="font-serif text-[15px] font-medium tracking-[-0.01em] text-[var(--foreground)]">
          {t("title")}{" "}
          <span className="italic text-[var(--primary)]">
            {t("titleAccent")}
          </span>
        </h3>
        <button
          type="button"
          onClick={() => markAllAsRead.mutate()}
          disabled={markAllAsRead.isPending}
          className="flex shrink-0 items-center gap-1 rounded-full bg-[var(--primary-soft)] px-2.5 py-1 font-mono text-[10px] font-bold uppercase tracking-[0.08em] text-[var(--primary-deep)] transition-colors hover:bg-[var(--primary)] hover:text-white disabled:opacity-60"
        >
          <Check className="h-3 w-3" aria-hidden="true" />
          {t("markAllAsRead")}
        </button>
      </div>

      {/* List */}
      <div className="max-h-96 overflow-y-auto">
        {isLoading ? (
          <div className="space-y-1 p-2">
            {Array.from({ length: 5 }).map((_, i) => (
              <div
                key={i}
                className="h-16 animate-pulse rounded-xl bg-[var(--background)]"
              />
            ))}
          </div>
        ) : displayed.length === 0 ? (
          <div className="flex flex-col items-center gap-2 px-4 py-10 text-center">
            <span
              aria-hidden="true"
              className="flex h-10 w-10 items-center justify-center rounded-full bg-[var(--primary-soft)] text-[var(--primary)]"
            >
              <Bell className="h-4 w-4" strokeWidth={1.6} />
            </span>
            <p className="font-serif text-[14px] italic text-[var(--muted-foreground)]">
              {t("empty")}
            </p>
          </div>
        ) : (
          <div className="divide-y divide-[var(--border)]">
            {displayed.map((n) => (
              <NotificationItem
                key={n.id}
                notification={n}
                onClose={onClose}
              />
            ))}
          </div>
        )}
      </div>

      {/* Footer */}
      <div className="border-t border-[var(--border)]">
        <Link
          href="/notifications"
          onClick={onClose}
          className="block py-2.5 text-center font-mono text-[11px] font-bold uppercase tracking-[0.08em] text-[var(--primary)] transition-colors hover:bg-[var(--primary-soft)] hover:text-[var(--primary-deep)]"
        >
          {t("seeAll")}
        </Link>
      </div>
    </div>
  )
}
