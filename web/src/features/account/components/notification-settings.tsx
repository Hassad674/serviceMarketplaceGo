"use client"

import { useState, useEffect } from "react"
import { useTranslations } from "next-intl"
import { Loader2 } from "lucide-react"
import { cn } from "@/shared/lib/utils"
import {
  useNotificationPreferences,
  useUpdateNotificationPreferences,
  type NotificationPreference,
} from "../hooks/use-notification-preferences"

type Channel = "in_app" | "push" | "email"

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

function Toggle({ checked, onChange, disabled }: { checked: boolean; onChange: () => void; disabled?: boolean }) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      onClick={disabled ? undefined : onChange}
      disabled={disabled}
      className={cn(
        "relative inline-flex h-6 w-11 shrink-0 cursor-pointer items-center rounded-full transition-colors",
        checked ? "bg-rose-500" : "bg-slate-300 dark:bg-slate-600",
        disabled && "cursor-not-allowed opacity-50",
      )}
    >
      <span
        className={cn(
          "inline-block h-4 w-4 rounded-full bg-white shadow-sm transition-transform",
          checked ? "translate-x-6" : "translate-x-1",
        )}
      />
    </button>
  )
}

export function NotificationSettings() {
  const t = useTranslations("account")
  const { data: prefs, isLoading } = useNotificationPreferences()
  const updateMutation = useUpdateNotificationPreferences()
  const [localPrefs, setLocalPrefs] = useState<NotificationPreference[]>([])

  useEffect(() => {
    if (prefs) setLocalPrefs(prefs)
  }, [prefs])

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
        if (prefs) setLocalPrefs(prefs)
      },
    })
  }

  if (isLoading) {
    return (
      <div className="space-y-4">
        {Array.from({ length: 4 }).map((_, i) => (
          <div key={i} className="h-24 animate-pulse rounded-xl bg-slate-100 dark:bg-slate-800" />
        ))}
      </div>
    )
  }

  const prefsMap = new Map(localPrefs.map((p) => [p.type, p]))

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-lg font-semibold text-slate-900 dark:text-slate-100">
          {t("notificationPreferences")}
        </h2>
        <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">
          {t("notificationPreferencesDesc")}
        </p>
      </div>

      {updateMutation.isError && (
        <div className="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-800 dark:bg-red-900/20 dark:text-red-400">
          {t("saveError")}
        </div>
      )}

      {GROUPS.map((group) => (
        <div
          key={group.titleKey}
          className="rounded-xl border border-slate-200 bg-white dark:border-slate-700 dark:bg-slate-800"
        >
          <div className="border-b border-slate-100 px-5 py-3.5 dark:border-slate-700">
            <h3 className="text-sm font-semibold text-slate-900 dark:text-slate-100">
              {t(group.titleKey)}
            </h3>
          </div>

          {/* Header */}
          <div className="hidden items-center gap-4 border-b border-slate-100 px-5 py-2 sm:flex dark:border-slate-700">
            <span className="flex-1 text-xs font-medium uppercase tracking-wider text-slate-400">
              {t("notificationType")}
            </span>
            <span className="w-16 text-center text-xs font-medium uppercase tracking-wider text-slate-400">
              {t("inApp")}
            </span>
            <span className="w-16 text-center text-xs font-medium uppercase tracking-wider text-slate-400">
              {t("push")}
            </span>
            <span className="w-16 text-center text-xs font-medium uppercase tracking-wider text-slate-400">
              {t("emailChannel")}
            </span>
          </div>

          {/* Rows */}
          <div className="divide-y divide-slate-100 dark:divide-slate-700">
            {group.types.map((type) => {
              const pref = prefsMap.get(type)
              if (!pref) return null
              const isNewMessage = type === "new_message"

              return (
                <div key={type} className="flex flex-col gap-3 px-5 py-3.5 sm:flex-row sm:items-center sm:gap-4">
                  <span className="flex-1 text-sm text-slate-700 dark:text-slate-300">
                    {t(TYPE_LABELS[type] || type)}
                  </span>
                  <div className="flex items-center gap-6 sm:gap-4">
                    <div className="flex items-center gap-2 sm:w-16 sm:justify-center">
                      <span className="text-xs text-slate-400 sm:hidden">{t("inApp")}</span>
                      <Toggle checked={pref.in_app} onChange={() => handleToggle(type, "in_app")} />
                    </div>
                    <div className="flex items-center gap-2 sm:w-16 sm:justify-center">
                      <span className="text-xs text-slate-400 sm:hidden">{t("push")}</span>
                      <Toggle checked={pref.push} onChange={() => handleToggle(type, "push")} />
                    </div>
                    <div className="flex items-center gap-2 sm:w-16 sm:justify-center">
                      <span className="text-xs text-slate-400 sm:hidden">{t("emailChannel")}</span>
                      <Toggle
                        checked={pref.email}
                        onChange={() => handleToggle(type, "email")}
                        disabled={isNewMessage}
                      />
                    </div>
                  </div>
                </div>
              )
            })}
          </div>
        </div>
      ))}

      {updateMutation.isPending && (
        <div className="flex items-center gap-2 text-sm text-slate-500">
          <Loader2 className="h-4 w-4 animate-spin" />
          {t("saving")}
        </div>
      )}
    </div>
  )
}
