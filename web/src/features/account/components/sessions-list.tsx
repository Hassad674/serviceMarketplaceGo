"use client"

import {
  HelpCircle,
  Loader2,
  Monitor,
  ShieldAlert,
  Smartphone,
  Tablet,
} from "lucide-react"
import { useTranslations, useLocale } from "next-intl"
import { useState } from "react"
import {
  useRevokeOtherSessions,
  useRevokeSession,
  useSessions,
} from "../hooks/use-sessions"
import type { Session } from "../api/sessions-api"
import { Button } from "@/shared/components/ui/button"

/**
 * SessionsList — Malt-style "Sessions actives" panel.
 *
 * Each row shows: device label · localisation · timestamp précis (FR
 * formatter) · IP /24. The current session is decorated with a
 * "Cette session" badge and its "Révoquer" button is hidden (we
 * deliberately keep the "revoke others" CTA as the single way to
 * log out other tabs/devices — revoking the current session from
 * here is confusing UX because it does not log the page itself out
 * until the access token expires).
 *
 * Optimistic update: revoking a session removes the row immediately;
 * a failure rolls the cache back and surfaces a toast (toast not
 * implemented here — the parent SecuritySettings will wire it up via
 * the existing toast plumbing).
 */
export function SessionsList() {
  const t = useTranslations("account.security.sessions")
  const locale = useLocale()
  const sessions = useSessions()
  const revokeOne = useRevokeSession()
  const revokeOthers = useRevokeOtherSessions()
  const [pendingId, setPendingId] = useState<string | null>(null)

  const rows: Session[] = sessions.data?.data ?? []
  const hasOthers = rows.some((s) => !s.is_current)

  return (
    <section
      aria-labelledby="active-sessions-heading"
      className="rounded-2xl border border-border bg-card p-6 shadow-[var(--shadow-card)]"
    >
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0">
          <h3
            id="active-sessions-heading"
            className="text-sm font-semibold uppercase tracking-[0.08em] text-muted-foreground"
          >
            {t("title")}
          </h3>
          <p className="mt-1.5 text-sm text-muted-foreground">{t("subtitle")}</p>
        </div>
        {hasOthers && (
          <Button
            type="button"
            variant="outline"
            size="sm"
            disabled={revokeOthers.isPending}
            onClick={() => {
              if (window.confirm(t("revokeOthersConfirm"))) {
                revokeOthers.mutate()
              }
            }}
          >
            {revokeOthers.isPending ? t("revoking") : t("revokeOthers")}
          </Button>
        )}
      </div>

      {sessions.isLoading ? (
        <SessionsSkeleton />
      ) : sessions.isError ? (
        <SessionsError onRetry={() => sessions.refetch()} />
      ) : rows.length === 0 ? (
        <SessionsEmpty />
      ) : (
        <ul className="mt-5 space-y-3">
          {rows.map((s) => (
            <SessionRow
              key={s.id}
              session={s}
              locale={locale}
              isRevoking={pendingId === s.id && revokeOne.isPending}
              onRevoke={() => {
                setPendingId(s.id)
                revokeOne.mutate(s.id)
              }}
            />
          ))}
        </ul>
      )}
    </section>
  )
}

function SessionRow({
  session,
  locale,
  isRevoking,
  onRevoke,
}: {
  session: Session
  locale: string
  isRevoking: boolean
  onRevoke: () => void
}) {
  const t = useTranslations("account.security.sessions")
  const Icon = iconFor(session.os)
  const created = formatPreciseDateTime(session.created_at, locale)
  const lastUsed = formatPreciseDateTime(session.last_used_at, locale)
  const location = formatLocation(session.city, session.country_code, t("unknownLocation"))

  return (
    <li className="flex items-start gap-3 rounded-xl border border-border bg-[var(--background)] p-4">
      <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-primary-soft text-[var(--primary-deep)]">
        <Icon className="h-[18px] w-[18px]" strokeWidth={1.6} aria-hidden="true" />
      </div>

      <div className="min-w-0 flex-1">
        <div className="flex flex-wrap items-center gap-2">
          <p className="text-sm font-medium text-foreground">{session.device_label}</p>
          {session.is_current && (
            <span className="inline-flex items-center rounded-full bg-primary-soft px-2 py-[2px] text-[11px] font-semibold uppercase tracking-[0.06em] text-[var(--primary-deep)]">
              {t("currentBadge")}
            </span>
          )}
        </div>
        <p className="mt-0.5 text-xs text-muted-foreground">
          {location}
          {session.ip_anonymized ? (
            <>
              {" · "}
              <span className="font-mono text-[12px] text-foreground">
                {session.ip_anonymized}
              </span>
            </>
          ) : null}
        </p>
        <p className="mt-1 text-[11px] uppercase tracking-[0.06em] text-muted-foreground">
          {created}
          {lastUsed && lastUsed !== created ? <> · {lastUsed}</> : null}
        </p>
      </div>

      {!session.is_current && (
        <Button
          type="button"
          variant="ghost"
          size="sm"
          disabled={isRevoking}
          onClick={onRevoke}
        >
          {isRevoking ? t("revoking") : t("revoke")}
        </Button>
      )}
    </li>
  )
}

function SessionsSkeleton() {
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

function SessionsEmpty() {
  const t = useTranslations("account.security.sessions")
  return (
    <div className="mt-6 rounded-xl border border-dashed border-border bg-[var(--background)] p-8 text-center">
      <p className="text-sm text-muted-foreground">{t("empty")}</p>
    </div>
  )
}

function SessionsError({ onRetry }: { onRetry: () => void }) {
  const t = useTranslations("account.security.sessions")
  const retryLabel = useTranslations("account.security")("retry")
  return (
    <div
      role="alert"
      className="mt-6 flex flex-col items-center gap-3 rounded-xl border border-dashed border-border bg-[var(--background)] p-6 text-center"
    >
      <ShieldAlert className="h-6 w-6 text-muted-foreground" aria-hidden="true" />
      <p className="text-sm text-muted-foreground">{t("error")}</p>
      <Button type="button" variant="outline" size="sm" onClick={onRetry}>
        {retryLabel}
      </Button>
    </div>
  )
}

function iconFor(os?: string) {
  switch ((os || "").toLowerCase()) {
    case "ios":
      return Smartphone
    case "android":
      return Smartphone
    case "ipados":
      return Tablet
    case "windows":
    case "macos":
    case "linux":
      return Monitor
    default:
      return HelpCircle
  }
}

// formatPreciseDateTime — Malt-style "11/05/2026 — 10:48:46" output
// using the user's locale for the day-month-year separator. We force
// 2-digit day/month/hour/minute/second so the row width stays
// predictable across locales.
function formatPreciseDateTime(iso: string, locale: string): string {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  try {
    return new Intl.DateTimeFormat(locale, {
      day: "2-digit",
      month: "2-digit",
      year: "numeric",
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit",
      hour12: false,
    }).format(d)
  } catch {
    return d.toLocaleString()
  }
}

function formatLocation(city?: string, country?: string, fallback?: string): string {
  const parts = [city, country].filter((p): p is string => Boolean(p && p.trim()))
  if (parts.length === 0) return fallback ?? ""
  return parts.join(" · ")
}

export const SessionsLoadingSpinner = () => (
  <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
)
