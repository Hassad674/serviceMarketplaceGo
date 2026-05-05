"use client"

import { useState } from "react"
import { useTranslations } from "next-intl"
import { Loader2, MailX, Mail } from "lucide-react"
import { cn } from "@/shared/lib/utils"
import {
  useNotificationPreferences,
  useUpdateNotificationPreferences,
  useBulkEmailPreferences,
  type NotificationPreference,
} from "../hooks/use-notification-preferences"

import { Button } from "@/shared/components/ui/button"

type Channel = "push" | "email"

const GROUPS = [
  {
    titleKey: "groupProposals",
    types: [
      "proposal_received",
      "proposal_accepted",
      "proposal_declined",
      "proposal_modified",
      "proposal_paid",
      "completion_requested",
      "proposal_completed",
    ],
  },
  {
    titleKey: "groupReviews",
    types: ["review_received"],
  },
  {
    titleKey: "groupMessages",
    types: ["new_message"],
  },
  {
    titleKey: "groupSystem",
    types: ["system_announcement"],
  },
]

const TYPE_LABELS: Record<string, string> = {
  proposal_received: "proposalReceived",
  proposal_accepted: "proposalAccepted",
  proposal_declined: "proposalDeclined",
  proposal_modified: "proposalModified",
  proposal_paid: "proposalPaid",
  completion_requested: "completionRequested",
  proposal_completed: "proposalCompleted",
  review_received: "reviewReceived",
  new_message: "newMessage",
  system_announcement: "systemAnnouncement",
}

/**
 * Soleil v2 toggle pill — 36×20 track with white knob, corail when on,
 * ivoire-soft (sable clair `--color-border`) when off. Mirrors the JSX
 * source of-truth (soleil-lotE.jsx, `Toggle` uses `SE.border` off /
 * `SE.accent` on).
 */
function Toggle({
  checked,
  onChange,
  disabled,
  ariaLabel,
}: {
  checked: boolean
  onChange: () => void
  disabled?: boolean
  ariaLabel?: string
}) {
  return (
    <Button
      variant="ghost"
      size="auto"
      type="button"
      role="switch"
      aria-checked={checked}
      aria-label={ariaLabel}
      onClick={disabled ? undefined : onChange}
      disabled={disabled}
      className={cn(
        "relative inline-flex h-5 w-9 shrink-0 items-center rounded-full p-0 transition-colors duration-150",
        checked
          ? "bg-primary hover:bg-primary"
          : "bg-border hover:bg-border",
        disabled && "cursor-not-allowed opacity-50",
      )}
    >
      <span
        className={cn(
          "inline-block h-4 w-4 rounded-full bg-white shadow-[0_1px_3px_rgba(0,0,0,0.18)] transition-transform duration-150",
          checked ? "translate-x-[18px]" : "translate-x-0.5",
        )}
      />
    </Button>
  )
}

interface BulkEmailToggleProps {
  allDisabled: boolean
  onToggle: () => void
  isPending: boolean
  t: ReturnType<typeof useTranslations<"account">>
}

/**
 * Soleil v2 bulk-email row card — corail-soft icon square, label +
 * helper copy, toggle on the right. Matches the JSX source's "global"
 * card right under the section heading.
 */
function BulkEmailToggle({ allDisabled, onToggle, isPending, t }: BulkEmailToggleProps) {
  const Icon = allDisabled ? MailX : Mail
  return (
    <div
      className={cn(
        "flex items-center gap-3.5 rounded-2xl border bg-card px-[18px] py-4 shadow-[var(--shadow-card)] transition-colors",
        allDisabled ? "border-[var(--warning)]/40" : "border-border",
      )}
    >
      <div
        className={cn(
          "flex h-9 w-9 shrink-0 items-center justify-center rounded-[10px]",
          allDisabled
            ? "bg-[var(--warning)]/15 text-[var(--warning)]"
            : "bg-primary-soft text-primary",
        )}
      >
        <Icon className="h-4 w-4" aria-hidden="true" />
      </div>
      <div className="flex-1 min-w-0">
        <p className="text-sm font-semibold text-foreground">
          {allDisabled ? t("bulkEmailDisabledLabel") : t("bulkEmailEnabledLabel")}
        </p>
        <p className="mt-0.5 text-xs text-muted-foreground">
          {t("bulkEmailDesc")}
        </p>
      </div>
      <Toggle
        checked={!allDisabled}
        onChange={onToggle}
        disabled={isPending}
        ariaLabel={t("bulkEmailEnabledLabel")}
      />
    </div>
  )
}

export function NotificationSettings() {
  const t = useTranslations("account")
  const { data: prefsResponse, isLoading } = useNotificationPreferences()
  const updateMutation = useUpdateNotificationPreferences()
  const bulkEmailMutation = useBulkEmailPreferences()
  const [localPrefs, setLocalPrefs] = useState<NotificationPreference[]>([])
  const [emailEnabled, setEmailEnabled] = useState(true)
  // Track which prefsResponse identity we have already mirrored into
  // local state. Comparing identity inside render lets us call setState
  // *during* render — React's "set state during render" pattern — which
  // is the recommended replacement for setState-in-effect when the only
  // job is to keep local optimistic state in sync with server data.
  // https://react.dev/reference/react/useState#storing-information-from-previous-renders
  const [syncedPrefs, setSyncedPrefs] = useState<typeof prefsResponse | null>(
    null,
  )
  if (prefsResponse && prefsResponse !== syncedPrefs) {
    setSyncedPrefs(prefsResponse)
    setLocalPrefs(prefsResponse.data)
    setEmailEnabled(prefsResponse.email_notifications_enabled)
  }

  const allEmailsDisabled = !emailEnabled

  function handleBulkEmailToggle() {
    const newEnabled = !emailEnabled
    setEmailEnabled(newEnabled)
    bulkEmailMutation.mutate(newEnabled, {
      onError: () => {
        if (prefsResponse) {
          setEmailEnabled(prefsResponse.email_notifications_enabled)
        }
      },
    })
  }

  function handleToggle(type: string, channel: Channel) {
    const updated = localPrefs.map((p) => {
      if (p.type === type) {
        return { ...p, [channel]: !p[channel] }
      }
      return p
    })
    setLocalPrefs(updated)
    updateMutation.mutate(updated, {
      onError: () => {
        if (prefsResponse) setLocalPrefs(prefsResponse.data)
      },
    })
  }

  if (isLoading) {
    return (
      <div className="space-y-4">
        {Array.from({ length: 4 }).map((_, i) => (
          <div
            key={i}
            className="h-24 animate-pulse rounded-2xl border border-border bg-card"
          />
        ))}
      </div>
    )
  }

  const prefsMap = new Map(localPrefs.map((p) => [p.type, p]))

  return (
    <div className="space-y-[18px]">
      <header className="mb-[22px]">
        <h2 className="font-serif text-[22px] font-semibold tracking-[-0.01em] text-foreground">
          {t("notificationPreferences")}
        </h2>
        <p className="mt-1.5 text-sm text-muted-foreground">
          {t("notificationPreferencesDesc")}
        </p>
      </header>

      {(updateMutation.isError || bulkEmailMutation.isError) && (
        <div
          role="alert"
          className="rounded-xl border border-[var(--destructive)]/40 bg-[var(--destructive)]/10 px-4 py-3 text-sm text-[var(--destructive)]"
        >
          {t("saveError")}
        </div>
      )}

      <BulkEmailToggle
        allDisabled={allEmailsDisabled}
        onToggle={handleBulkEmailToggle}
        isPending={bulkEmailMutation.isPending}
        t={t}
      />

      {GROUPS.map((group) => (
        <NotificationGroupCard
          key={group.titleKey}
          titleKey={group.titleKey}
          types={group.types}
          prefsMap={prefsMap}
          allEmailsDisabled={allEmailsDisabled}
          onToggle={handleToggle}
          t={t}
        />
      ))}

      {(updateMutation.isPending || bulkEmailMutation.isPending) && (
        <div className="flex items-center gap-2 text-sm text-muted-foreground">
          <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
          {t("saving")}
        </div>
      )}
    </div>
  )
}

interface NotificationGroupCardProps {
  titleKey: string
  types: string[]
  prefsMap: Map<string, NotificationPreference>
  allEmailsDisabled: boolean
  onToggle: (type: string, channel: Channel) => void
  t: ReturnType<typeof useTranslations<"account">>
}

/**
 * One Soleil v2 group card: bold title row + uppercase mono header row
 * (Type / Push / Email) + dividers between rows. Matches the source.
 */
function NotificationGroupCard({
  titleKey,
  types,
  prefsMap,
  allEmailsDisabled,
  onToggle,
  t,
}: NotificationGroupCardProps) {
  return (
    <section className="overflow-hidden rounded-2xl border border-border bg-card shadow-[var(--shadow-card)]">
      <header className="border-b border-border px-5 py-3.5">
        <h3 className="text-sm font-bold text-foreground">{t(titleKey)}</h3>
      </header>

      {/* Header row (visible from sm+ to keep mobile compact). The
          "Type" cell is intentionally empty — the row label below is
          self-explanatory and the duplicate "TYPE" header was visual
          noise. */}
      <div className="hidden grid-cols-[1fr_70px_70px] items-center gap-2 border-b border-border bg-[var(--background)] px-5 py-2 sm:grid">
        <span aria-hidden="true" />
        <span className="text-center text-[11px] font-semibold uppercase tracking-[0.06em] text-muted-foreground">
          {t("push")}
        </span>
        <span className="text-center text-[11px] font-semibold uppercase tracking-[0.06em] text-muted-foreground">
          {t("emailChannel")}
        </span>
      </div>

      <div className="divide-y divide-border">
        {types.map((type) => {
          const pref = prefsMap.get(type)
          if (!pref) return null
          const isNewMessage = type === "new_message"
          const label = t(TYPE_LABELS[type] || type)

          return (
            <div
              key={type}
              className="flex flex-col gap-3 px-5 py-3 sm:grid sm:grid-cols-[1fr_70px_70px] sm:items-center sm:gap-2"
            >
              <span className="text-[13.5px] text-foreground">{label}</span>
              <div className="flex items-center gap-6 sm:contents">
                <div className="flex items-center gap-2 sm:justify-center">
                  <span className="text-xs text-muted-foreground sm:hidden">
                    {t("push")}
                  </span>
                  <Toggle
                    checked={pref.push}
                    onChange={() => onToggle(type, "push")}
                    ariaLabel={`${label} — ${t("push")}`}
                  />
                </div>
                <div className="flex items-center gap-2 sm:justify-center">
                  <span className="text-xs text-muted-foreground sm:hidden">
                    {t("emailChannel")}
                  </span>
                  <Toggle
                    checked={allEmailsDisabled ? false : pref.email}
                    onChange={() => onToggle(type, "email")}
                    disabled={isNewMessage || allEmailsDisabled}
                    ariaLabel={`${label} — ${t("emailChannel")}`}
                  />
                </div>
              </div>
            </div>
          )
        })}
      </div>
    </section>
  )
}
