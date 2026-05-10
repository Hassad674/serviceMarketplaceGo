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
// drives three children: visibility line chart, search appearances
// line chart, and the keywords table. /stats is gated to Provider /
// Agency on the page-level shell — Enterprise + Referrer redirect to
// /dashboard upstream.

const ALLOWED_PERIODS: StatsPeriodDays[] = [7, 30, 90]
const DEFAULT_PERIOD: StatsPeriodDays = 30
const KEYWORDS_LIMIT = 10

export function StatsOverview() {
  const t = useTranslations("stats")
  const tEmpty = useTranslations("stats.empty")
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
  const hasData = series.some((point) => point.count > 0)
  const positionSeries = useMemo(
    () => buildPositionSeries(visibility.data?.avg_search_position, series),
    [visibility.data?.avg_search_position, series],
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
      {!hasData && !visibility.isLoading ? (
        <p className="rounded-2xl border border-dashed border-border bg-card p-6 text-center text-sm text-muted-foreground">
          {tEmpty("notEnoughData")}
        </p>
      ) : null}
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <LineChart
          series={series}
          title={t("charts.profileViews")}
          emptyMessage={tEmpty("notEnoughData")}
        />
        <LineChart
          series={series}
          title={t("charts.searchAppearances")}
          emptyMessage={tEmpty("notEnoughData")}
          className="text-primary-deep"
        />
      </div>
      <LineChart
        series={positionSeries}
        title={t("charts.avgPosition")}
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
