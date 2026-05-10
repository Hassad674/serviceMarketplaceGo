"use client"

import {
  Loader2,
  Monitor,
  Smartphone,
  Tablet,
  HelpCircle,
  ShieldAlert,
} from "lucide-react"
import { useTranslations, useLocale } from "next-intl"
import { useSecurityActivity } from "../hooks/use-security-activity"
import { Button } from "@/shared/components/ui/button"
import type {
  SecurityActivityEvent,
} from "../api/security-api"
import { TwoFactorToggle } from "./two-factor-toggle"
import { useUser } from "@/shared/hooks/use-user"

/**
 * SecuritySettings — Soleil v2 panel for the /account?section=security
 * tab.
 *
 * Lists the caller's recent authentication events (login, logout,
 * token refresh, password reset request/complete) so the user can
 * spot a session that does not look like theirs and react fast.
 *
 * Empty / error states are first-class — every async surface the
 * marketplace ships has them and this page is no exception.
 */
export function SecuritySettings() {
  const t = useTranslations("account.security")
  const locale = useLocale()
  const query = useSecurityActivity()
  const { data: user } = useUser()

  const events: SecurityActivityEvent[] = query.data
    ? query.data.pages.flatMap((p) => p.data)
    : []

  return (
    <div className="space-y-6">
      <header>
        <h2 className="font-serif text-[22px] font-semibold tracking-[-0.01em] text-foreground">
          {t("title")}
        </h2>
        <p className="mt-1.5 text-sm text-muted-foreground">{t("subtitle")}</p>
      </header>

      {/* B.6 — Email 2FA toggle. The flag is not exposed on /auth/me yet
          so we default to false; on the next page load the parent will
          re-read once the field is wired (flagged for follow-up). */}
      <TwoFactorToggle initialEnabled={user?.two_factor_email_enabled ?? false} />

      <section
        aria-labelledby="security-activity-heading"
        className="rounded-2xl border border-border bg-card p-6 shadow-[var(--shadow-card)]"
      >
        <h3
          id="security-activity-heading"
          className="text-sm font-semibold uppercase tracking-[0.08em] text-muted-foreground"
        >
          {t("recentActivity")}
        </h3>

        {query.isLoading ? (
          <SecurityListSkeleton />
        ) : query.isError ? (
          <SecurityErrorState onRetry={() => query.refetch()} />
        ) : events.length === 0 ? (
          <SecurityEmptyState />
        ) : (
          <SecurityEventList events={events} locale={locale} />
        )}

        {query.hasNextPage && (
          <div className="mt-5 flex justify-center">
            <Button
              type="button"
              variant="outline"
              size="md"
              disabled={query.isFetchingNextPage}
              onClick={() => query.fetchNextPage()}
            >
              {query.isFetchingNextPage ? t("loadingMore") : t("loadMore")}
            </Button>
          </div>
        )}
      </section>
    </div>
  )
}

function SecurityListSkeleton() {
  return (
    <ul className="mt-5 space-y-3" aria-busy="true">
      {[0, 1, 2].map((i) => (
        <li
          key={i}
          className="flex animate-pulse items-center gap-3 rounded-xl border border-border bg-[var(--background)] p-4"
        >
          <div className="h-9 w-9 rounded-lg bg-border" />
          <div className="flex-1 space-y-2">
            <div className="h-3 w-1/2 rounded-full bg-border" />
            <div className="h-3 w-1/3 rounded-full bg-border" />
          </div>
        </li>
      ))}
    </ul>
  )
}

function SecurityEmptyState() {
  const t = useTranslations("account.security")
  return (
    <div className="mt-6 rounded-xl border border-dashed border-border bg-[var(--background)] p-8 text-center">
      <p className="text-sm text-muted-foreground">{t("empty")}</p>
    </div>
  )
}

function SecurityErrorState({ onRetry }: { onRetry: () => void }) {
  const t = useTranslations("account.security")
  return (
    <div
      role="alert"
      className="mt-6 flex flex-col items-center gap-3 rounded-xl border border-dashed border-border bg-[var(--background)] p-6 text-center"
    >
      <ShieldAlert className="h-6 w-6 text-muted-foreground" aria-hidden="true" />
      <p className="text-sm text-muted-foreground">{t("error")}</p>
      <Button type="button" variant="outline" size="sm" onClick={onRetry}>
        {t("retry")}
      </Button>
    </div>
  )
}

function SecurityEventList({
  events,
  locale,
}: {
  events: SecurityActivityEvent[]
  locale: string
}) {
  return (
    <ul className="mt-5 space-y-3">
      {events.map((event) => (
        <SecurityEventRow key={event.id} event={event} locale={locale} />
      ))}
    </ul>
  )
}

function SecurityEventRow({
  event,
  locale,
}: {
  event: SecurityActivityEvent
  locale: string
}) {
  const t = useTranslations("account.security")
  const Icon = iconFor(event.access_kind)
  const dateText = formatDateTime(event.created_at, locale)
  const deviceLabel = event.user_agent_summary || t("unknownDevice")
  const actionLabel = actionLabelKey(event.action)
  const ip = event.ip_address || ""

  return (
    <li className="flex items-start gap-3 rounded-xl border border-border bg-[var(--background)] p-4">
      <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-primary-soft text-[var(--primary-deep)]">
        <Icon className="h-[18px] w-[18px]" strokeWidth={1.6} aria-hidden="true" />
      </div>
      <div className="min-w-0 flex-1">
        <p className="text-sm font-medium text-foreground">{deviceLabel}</p>
        <p className="mt-0.5 text-xs text-muted-foreground">
          {t(actionLabel)}
          {ip ? (
            <>
              {" · "}
              <span className="font-mono text-[12px] text-foreground">{ip}</span>
            </>
          ) : null}
          {event.country_hint ? (
            <>
              {" · "}
              <span>{event.country_hint}</span>
            </>
          ) : null}
        </p>
      </div>
      <time
        dateTime={event.created_at}
        className="shrink-0 whitespace-nowrap text-xs text-muted-foreground"
      >
        {dateText}
      </time>
    </li>
  )
}

function iconFor(kind: SecurityActivityEvent["access_kind"]) {
  switch (kind) {
    case "desktop":
      return Monitor
    case "mobile":
      return Smartphone
    case "tablet":
      return Tablet
    default:
      return HelpCircle
  }
}

const ACTION_LABEL_KEYS: Record<string, string> = {
  "auth.login_success": "actions.loginSuccess",
  "auth.logout": "actions.logout",
  "auth.token_refresh": "actions.tokenRefresh",
  "auth.password_reset_request": "actions.passwordResetRequest",
  "auth.password_reset_complete": "actions.passwordResetComplete",
}

function actionLabelKey(action: string): string {
  return ACTION_LABEL_KEYS[action] ?? "actions.unknown"
}

function formatDateTime(iso: string, locale: string): string {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  try {
    return new Intl.DateTimeFormat(locale, {
      day: "2-digit",
      month: "short",
      hour: "2-digit",
      minute: "2-digit",
    }).format(d)
  } catch {
    return d.toLocaleString()
  }
}

// Re-export the spinner so the consumer can render a focused loading
// state if it ever wants to. Currently unused in this file, kept as
// a tiny utility for parity with the rest of the account/ surface.
export const SecurityLoadingSpinner = () => (
  <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
)
