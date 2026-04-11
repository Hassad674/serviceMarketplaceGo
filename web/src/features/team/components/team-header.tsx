"use client"

import { Users2, Building2 } from "lucide-react"
import { useTranslations } from "next-intl"
import type { CurrentOrganization } from "@/shared/hooks/use-user"

// Summary card at the top of /team: org type + counts. Pure
// presentational — the numbers come from the parent after it has
// loaded members + invitations.

type TeamHeaderProps = {
  organization: CurrentOrganization
  memberCount: number
  pendingInvitationCount: number
}

export function TeamHeader({
  organization,
  memberCount,
  pendingInvitationCount,
}: TeamHeaderProps) {
  const t = useTranslations("team")

  const typeLabel =
    organization.type === "agency" ? t("agencyType") : t("enterpriseType")

  return (
    <div className="rounded-xl border border-gray-100 dark:border-slate-700 bg-white dark:bg-slate-800 p-6">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div className="flex items-start gap-4">
          <div className="rounded-xl bg-rose-50 dark:bg-rose-500/10 p-3">
            <Building2 className="h-6 w-6 text-rose-500" />
          </div>
          <div>
            <h1 className="text-xl font-bold text-gray-900 dark:text-white">
              {t("title")}
            </h1>
            <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {typeLabel}
            </p>
          </div>
        </div>

        <div className="flex flex-wrap gap-3">
          <StatPill
            icon={<Users2 className="h-4 w-4" />}
            label={t("membersCount", { count: memberCount })}
          />
          {pendingInvitationCount > 0 && (
            <StatPill
              label={t("pendingInvitationsCount", { count: pendingInvitationCount })}
              highlight
            />
          )}
        </div>
      </div>
    </div>
  )
}

type StatPillProps = {
  icon?: React.ReactNode
  label: string
  highlight?: boolean
}

function StatPill({ icon, label, highlight }: StatPillProps) {
  return (
    <div
      className={
        highlight
          ? "inline-flex items-center gap-1.5 rounded-full bg-amber-50 dark:bg-amber-500/10 px-3 py-1.5 text-sm font-medium text-amber-700 dark:text-amber-300"
          : "inline-flex items-center gap-1.5 rounded-full bg-gray-100 dark:bg-slate-700 px-3 py-1.5 text-sm font-medium text-gray-700 dark:text-gray-300"
      }
    >
      {icon}
      {label}
    </div>
  )
}
