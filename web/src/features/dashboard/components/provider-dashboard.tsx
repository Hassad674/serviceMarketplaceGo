"use client"

import { Briefcase, Eye, Search, TrendingUp, ChevronRight } from "lucide-react"
import { useTranslations } from "next-intl"
import { StatCard } from "./widgets/stat-card"
import { ActionsTodoCard } from "./widgets/actions-todo-card"
import { SectionCard } from "./widgets/section-card"
import { Link } from "@i18n/navigation"
import type { DashboardAction } from "../types"
import type { VisibilityStats } from "@/features/stats/api/stats-api"

// ProviderDashboard is the layout shipped to Agency + Provider roles
// (excluding referrer mode). All data flows in via props — the
// page-level orchestrator (`app/[locale]/(app)/dashboard/page.tsx`)
// owns the TanStack Query reads and threads the slices through here.
//
// Stat strip (top, 4 cards):
//   1. Profile views (7 days, with sparkline + delta)
//   2. Search appearances (7 days, with sparkline)
//   3. Average search position (no sparkline — value is a single
//      derived number; trend over 7 days is too noisy to show)
//   4. Monthly revenue placeholder (pulled from existing wallet data
//      by the parent — defaults to "—" until the wallet feature
//      exposes a stable hook)
//
// Below the fold:
//   * Actions à faire (left, full-width on mobile / half on desktop)
//   * Pipeline / Active work (right)
//   * "Voir mes statistiques détaillées →" CTA pointing at /stats

export interface ProviderDashboardProps {
  visibilityStats: VisibilityStats | undefined
  isVisibilityLoading: boolean
  monthlyRevenueLabel: string
  pipelineCount: number
  pipelineCtaHref: string
  actions: DashboardAction[]
  actionsLoading: boolean
}

export function ProviderDashboard(props: ProviderDashboardProps) {
  const t = useTranslations("dashboard")
  const tStats = useTranslations("dashboard.stats")
  const series = props.visibilityStats?.series.map((p) => p.count) ?? []
  const totalViews = props.visibilityStats?.total_views ?? 0
  const searchAppearances = props.visibilityStats?.search_appearances ?? 0
  const avgPosition = props.visibilityStats?.avg_search_position
  return (
    <div className="space-y-6">
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard
          icon={Eye}
          tone="primary"
          label={tStats("profileViews7d")}
          value={formatCount(totalViews)}
          subLabel={tStats("last7Days")}
          series={series}
          isLoading={props.isVisibilityLoading}
          hideDelta
        />
        <StatCard
          icon={Search}
          tone="deep"
          label={tStats("searchAppearances7d")}
          value={formatCount(searchAppearances)}
          subLabel={tStats("last7Days")}
          series={series}
          isLoading={props.isVisibilityLoading}
          hideDelta
        />
        <StatCard
          icon={TrendingUp}
          tone="amber"
          label={tStats("avgSearchPosition")}
          value={formatPosition(avgPosition)}
          subLabel={
            avgPosition === null || avgPosition === undefined
              ? tStats("notEnoughData")
              : tStats("acrossPeriod")
          }
          isLoading={props.isVisibilityLoading}
          hideDelta
        />
        <StatCard
          icon={Briefcase}
          tone="success"
          label={tStats("monthlyRevenue")}
          value={props.monthlyRevenueLabel}
          subLabel={tStats("thisMonth")}
          hideDelta
        />
      </div>
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <ActionsTodoCard actions={props.actions} isLoading={props.actionsLoading} />
        <SectionCard
          title={t("pipeline.title")}
          ctaLabel={t("pipeline.cta")}
          ctaHref={props.pipelineCtaHref}
          emptyMessage={t("pipeline.empty")}
          isEmpty={props.pipelineCount === 0}
        >
          {props.pipelineCount > 0 ? (
            <p className="text-sm text-muted-foreground">
              {t("pipeline.count", { count: props.pipelineCount })}
            </p>
          ) : null}
        </SectionCard>
      </div>
      <Link
        href="/stats"
        className="group inline-flex items-center gap-1 text-[13px] font-semibold text-primary hover:text-primary-deep focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-primary/15 focus-visible:rounded"
      >
        {tStats("viewDetailedStats")}
        <ChevronRight
          className="h-3.5 w-3.5 transition-transform group-hover:translate-x-0.5"
          aria-hidden
        />
      </Link>
    </div>
  )
}

function formatCount(n: number): string {
  if (n >= 1000) {
    return `${(n / 1000).toFixed(1)}k`
  }
  return String(n)
}

function formatPosition(p: number | null | undefined): string {
  if (typeof p !== "number" || Number.isNaN(p)) return "—"
  return p.toFixed(1)
}
