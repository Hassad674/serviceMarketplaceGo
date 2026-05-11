"use client"

import { useCallback, useMemo } from "react"
import { useRouter, useSearchParams } from "next/navigation"
import { useTranslations } from "next-intl"
import { useUser } from "@/shared/hooks/use-user"
import { useKeywordStats, useVisibilityStats } from "../hooks/use-stats"
import { LineChart } from "@/shared/components/charts/line-chart"
import { PeriodSelector } from "./period-selector"
import { KeywordsTable } from "./keywords-table"
import type { StatsPeriodDays } from "../api/stats-api"

// StatsOverview is the meaty client component behind the /stats page.
// Owns the period state (synchronised with `?period=` in the URL) and
// drives a top summary strip (unit counts for total views, search
// appearances, average position) followed by three line charts and
// the keywords table.
//
// Empty-state policy:
//   * total_views and search_appearances are UNIT counts — always
//     render as a number (including "0") with a small caption when 0.
//     Never gate them behind a "wait 7 days" copy.
//   * avg_search_position is statistical — when fewer than 10 search
//     appearances exist, render the patience caption instead of a
//     misleading average. Threshold is intentional: a single
//     appearance with a fluke position would otherwise be reported as
//     the long-run average.
//
// /stats is gated to Provider / Agency on the page-level shell —
// Enterprise + Referrer redirect to /dashboard upstream.

const ALLOWED_PERIODS: StatsPeriodDays[] = [7, 30, 90]
const DEFAULT_PERIOD: StatsPeriodDays = 30
const KEYWORDS_LIMIT = 10
const POSITION_STATISTICAL_SIGNIFICANCE = 10

export function StatsOverview() {
  const t = useTranslations("stats")
  const tMetrics = useTranslations("stats.metrics")
  const tEmpty = useTranslations("stats.empty")
  const tCharts = useTranslations("stats.charts")
  const router = useRouter()
  const searchParams = useSearchParams()
  const { data: user } = useUser()

  const period = readPeriod(searchParams.get("period"))
  const setPeriod = useCallback(
    (next: StatsPeriodDays) => {
      const params = new URLSearchParams(searchParams.toString())
      params.set("period", String(next))
      router.replace(`?${params.toString()}`, { scroll: false })
    },
    [router, searchParams],
  )

  const visibility = useVisibilityStats(period)
  const keywords = useKeywordStats(period, KEYWORDS_LIMIT)

  const series = useMemo(
    () => visibility.data?.series ?? [],
    [visibility.data?.series],
  )
  const totalViews = visibility.data?.total_views ?? 0
  const searchAppearances = visibility.data?.search_appearances ?? 0
  const avgPosition = visibility.data?.avg_search_position ?? null
  const hasEnoughForPosition =
    searchAppearances >= POSITION_STATISTICAL_SIGNIFICANCE &&
    typeof avgPosition === "number"

  const positionSeries = useMemo(
    () => buildPositionSeries(avgPosition, series),
    [avgPosition, series],
  )

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <p className="font-mono text-[11px] font-bold uppercase tracking-[0.12em] text-primary">
            {t("eyebrow")}
          </p>
          <h1 className="font-serif text-[28px] font-medium tracking-[-0.02em] text-foreground sm:text-[34px]">
            {t("title")}
          </h1>
          <p className="mt-1 max-w-xl text-sm text-muted-foreground">
            {t("subtitle")}
          </p>
        </div>
        <PeriodSelector value={period} onChange={setPeriod} />
      </div>
      {visibility.error ? (
        <ErrorPanel message={t("errorLoading")} />
      ) : null}
      <div
        className="grid grid-cols-1 gap-4 sm:grid-cols-3"
        data-testid="stats-metric-strip"
      >
        <MetricCard
          label={tMetrics("totalViewsLabel")}
          value={formatInteger(totalViews)}
          caption={
            totalViews === 0 ? tMetrics("totalViewsEmpty") : undefined
          }
          isLoading={visibility.isLoading}
        />
        <MetricCard
          label={tMetrics("searchAppearancesLabel")}
          value={formatInteger(searchAppearances)}
          caption={
            searchAppearances === 0
              ? tMetrics("searchAppearancesEmpty")
              : undefined
          }
          isLoading={visibility.isLoading}
        />
        <MetricCard
          label={tMetrics("avgPositionLabel")}
          value={
            hasEnoughForPosition
              ? tMetrics("avgPositionUnit", {
                  value: Math.round(avgPosition as number),
                })
              : "—"
          }
          caption={
            !hasEnoughForPosition
              ? tMetrics("avgPositionPatience")
              : undefined
          }
          isLoading={visibility.isLoading}
        />
      </div>
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <LineChart
          series={series}
          title={tCharts("profileViews")}
          emptyMessage={tCharts("emptyChart")}
        />
        <LineChart
          series={series}
          title={tCharts("searchAppearances")}
          emptyMessage={tCharts("emptyChart")}
          className="text-primary-deep"
        />
      </div>
      <LineChart
        series={positionSeries}
        title={tCharts("avgPosition")}
        emptyMessage={tEmpty("notEnoughPosition")}
      />
      <section aria-labelledby="top-keywords-heading">
        <header className="mb-3 flex items-center justify-between">
          <h2
            id="top-keywords-heading"
            className="font-serif text-[20px] font-medium text-foreground"
          >
            {t("topKeywords")}
          </h2>
          <span className="font-mono text-[11px] font-bold uppercase tracking-[0.12em] text-muted-foreground">
            {t("topKeywordsLimit", { count: KEYWORDS_LIMIT })}
          </span>
        </header>
        <KeywordsTable
          rows={keywords.data ?? []}
          isLoading={keywords.isLoading}
        />
      </section>
      {!user ? null : (
        <p className="text-xs text-muted-foreground">
          {t("footnote")}
        </p>
      )}
    </div>
  )
}

interface MetricCardProps {
  label: string
  value: string
  caption?: string
  isLoading?: boolean
}

// MetricCard renders a single "label + big number + optional caption"
// tile in the stats summary strip. Kept inline (not extracted to
// shared/) until a third use case justifies it — see the "rule of
// three" in CLAUDE.md. Local to the stats feature.
function MetricCard({ label, value, caption, isLoading }: MetricCardProps) {
  return (
    <div className="rounded-2xl border border-border bg-card p-5 shadow-card">
      <p className="font-mono text-[11px] font-bold uppercase tracking-[0.12em] text-muted-foreground">
        {label}
      </p>
      <p
        className={
          isLoading
            ? "mt-2 animate-pulse font-serif text-[30px] font-medium leading-tight tracking-[-0.02em] text-muted-foreground/40"
            : "mt-2 font-serif text-[30px] font-medium leading-tight tracking-[-0.02em] text-foreground"
        }
      >
        {isLoading ? "—" : value}
      </p>
      {caption ? (
        <p className="mt-1.5 text-[12px] text-muted-foreground">{caption}</p>
      ) : null}
    </div>
  )
}

function formatInteger(value: number): string {
  return new Intl.NumberFormat("fr-FR").format(value)
}

function readPeriod(raw: string | null): StatsPeriodDays {
  if (!raw) return DEFAULT_PERIOD
  const parsed = Number(raw)
  if ((ALLOWED_PERIODS as number[]).includes(parsed)) {
    return parsed as StatsPeriodDays
  }
  return DEFAULT_PERIOD
}

// buildPositionSeries projects the avg_search_position scalar onto the
// existing daily series by repeating the average across every bucket
// where search appearances occurred. Backend doesn't ship a per-day
// position series yet (would require a daily aggregation table), so
// this surfaces the average as a flat line — consumers see an honest
// "this is the period average" instead of a misleading per-day chart.
function buildPositionSeries(
  avg: number | null | undefined,
  series: { date: string; count: number }[],
): { date: string; count: number }[] {
  if (typeof avg !== "number" || series.length === 0) return []
  return series.map((point) => ({
    date: point.date,
    count: point.count > 0 ? avg : 0,
  }))
}

function ErrorPanel({ message }: { message: string }) {
  return (
    <div
      role="alert"
      className="rounded-2xl border border-primary/30 bg-primary-soft px-4 py-3 text-sm text-primary-deep"
    >
      {message}
    </div>
  )
}
