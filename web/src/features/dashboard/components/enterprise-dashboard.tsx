"use client"

import { Briefcase, FolderOpen, Inbox, Wallet } from "lucide-react"
import { useTranslations } from "next-intl"
import { StatCard } from "./widgets/stat-card"
import { ActionsTodoCard } from "./widgets/actions-todo-card"
import { SectionCard } from "./widgets/section-card"
import type { DashboardAction } from "../types"
import type { EnterpriseApplicationsStats } from "@/features/stats/api/stats-api"

// EnterpriseDashboard is the layout shipped to Enterprise organisations.
// It deliberately omits the "recommended providers" panel — the user
// explicitly refused that surface in the recap (R-DASH-2026-05-10).
// The four stat tiles cover the day-to-day operating numbers an
// Enterprise needs to triage from a single screen.

export interface EnterpriseDashboardProps {
  applicationsStats: EnterpriseApplicationsStats | undefined
  isApplicationsLoading: boolean
  activeRecruitments: number
  pendingProposals: number
  spendingLabel: string
  actions: DashboardAction[]
  actionsLoading: boolean
}

export function EnterpriseDashboard(props: EnterpriseDashboardProps) {
  const t = useTranslations("dashboard")
  const tStats = useTranslations("dashboard.stats")
  const series = props.applicationsStats?.series.map((p) => p.count) ?? []
  const totalApplications = props.applicationsStats?.total_count ?? 0
  return (
    <div className="space-y-6">
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard
          icon={FolderOpen}
          tone="primary"
          label={tStats("activeRecruitments")}
          value={String(props.activeRecruitments)}
          hideDelta
        />
        <StatCard
          icon={Inbox}
          tone="deep"
          label={tStats("applicationsReceived7d")}
          value={String(totalApplications)}
          subLabel={tStats("last7Days")}
          series={series}
          isLoading={props.isApplicationsLoading}
          hideDelta
        />
        <StatCard
          icon={Wallet}
          tone="amber"
          label={tStats("spending30d")}
          value={props.spendingLabel}
          subLabel={tStats("last30Days")}
          hideDelta
        />
        <StatCard
          icon={Briefcase}
          tone="success"
          label={tStats("toReview")}
          value={String(props.pendingProposals)}
          hideDelta
        />
      </div>
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <ActionsTodoCard actions={props.actions} isLoading={props.actionsLoading} />
        <SectionCard
          title={t("activeRecruitments.title")}
          ctaLabel={t("activeRecruitments.cta")}
          ctaHref="/jobs"
          emptyMessage={t("activeRecruitments.empty")}
          isEmpty={props.activeRecruitments === 0}
        >
          {props.activeRecruitments > 0 ? (
            <p className="text-sm text-muted-foreground">
              {t("activeRecruitments.count", { count: props.activeRecruitments })}
            </p>
          ) : null}
        </SectionCard>
      </div>
    </div>
  )
}
