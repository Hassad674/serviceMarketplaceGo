"use client"

import { useCallback, useMemo } from "react"
import { useRouter, useSearchParams } from "next/navigation"
import { useTranslations } from "next-intl"
import { useUser } from "@/shared/hooks/use-user"
import { useKeywordStats, useVisibilityStats } from "../hooks/use-stats"
import { LineChart } from "@/shared/components/charts/line-chart"
import { PeriodSelector } from "./period-selector"
import { KeywordsTable } from "./keywords-table"
import type { StatsPeriodDays, StatsTimeBucket } from "../api/stats-api"

// StatsOverview is the meaty client component behind the /stats page.
// Owns the period state (synchronised with `?period=` in the URL) and
// drives a top summary strip (unit counts for unique viewers, total
// views, search appearances) followed by line charts and the keywords
// table.
//
// D3 changes:
//   * Period selector now includes 365 (1 year) so the user can scan
//     a long-tail trend.
//   * Profile-views chart renders TWO lines — unique (corail solid,
//     primary) + total (corail dashed, secondary) — so the user can
//     see at a glance how many distinct people visited vs how often.
//   * Summary cards split into unique + total + search appearances so
//     the user understands the difference between the two view
//     counts.
//   * When the org has zero views across the period, the chart is
//     replaced by a friendly accentSoft empty card prompting a
//     LinkedIn share.
//
// /stats is gated to Provider / Agency on the page-level shell —
// Enterprise + Referrer redirect to /dashboard upstream.

const ALLOWED_PERIODS: StatsPeriodDays[] = [7, 30, 90, 365]
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

  const series: StatsTimeBucket[] = useMemo(
    () => visibility.data?.series ?? [],
    [visibility.data?.series],
  )
  const uniqueSeries = useMemo(() => seriesForKey(series, "unique"), [series])
  const totalSeries = useMemo(() => seriesForKey(series, "count"), [series])

  const totalViews = visibility.data?.total_views ?? 0
  const uniqueViewers = visibility.data?.unique_viewers ?? 0
  const searchAppearances = visibility.data?.search_appearances ?? 0
  const avgPosition = visibility.data?.avg_search_position ?? null
  const hasEnoughForPosition =
    searchAppearances >= POSITION_STATISTICAL_SIGNIFICANCE &&
    typeof avgPosition === "number"
  const hasAnyViews = totalViews > 0
  const isLoading = visibility.isLoading

  const positionSeries = useMemo(
    () => buildPositionSeries(avgPosition, totalSeries),
    [avgPosition, totalSeries],
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
        className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4"
        data-testid="stats-metric-strip"
      >
        <MetricCard
          label={tMetrics("uniqueViewersLabel")}
          value={formatInteger(uniqueViewers)}
          caption={tMetrics("totalViewsCaptionUnique")}
          isLoading={isLoading}
        />
        <MetricCard
          label={tMetrics("totalViewsLabel")}
          value={formatInteger(totalViews)}
          caption={
            totalViews === 0
              ? tMetrics("totalViewsEmpty")
              : tMetrics("totalViewsCaptionTotal")
          }
          isLoading={isLoading}
        />
        <MetricCard
          label={tMetrics("searchAppearancesLabel")}
          value={formatInteger(searchAppearances)}
          caption={
            searchAppearances === 0
              ? tMetrics("searchAppearancesEmpty")
              : undefined
          }
          isLoading={isLoading}
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
          isLoading={isLoading}
        />
      </div>
      {!isLoading && !hasAnyViews ? (
        <NoViewsCard
          title={tEmpty("noViewsTitle")}
          body={tEmpty("noViewsBody")}
        />
      ) : (
        <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
          <LineChart
            series={uniqueSeries}
            secondarySeries={totalSeries}
            title={tCharts("profileViews")}
            primaryLabel={tCharts("legendUnique")}
            secondaryLabel={tCharts("legendTotal")}
            emptyMessage={tCharts("emptyChart")}
          />
          <LineChart
            series={totalSeries}
            title={tCharts("searchAppearances")}
            emptyMessage={tCharts("emptyChart")}
            className="text-primary-deep"
          />
        </div>
      )}
      {hasAnyViews ? (
        <LineChart
          series={positionSeries}
          title={tCharts("avgPosition")}
          emptyMessage={tEmpty("notEnoughPosition")}
        />
      ) : null}
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
        <p className="text-xs text-muted-foreground">{t("footnote")}</p>
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

interface NoViewsCardProps {
  title: string
  body: string
}

// NoViewsCard renders the empty state when the org has zero recorded
// views across the selected period. Uses the Soleil v2 `accentSoft`
// background to soften the message — the user complaint in D3 was
// that the previous empty state looked alarming.
function NoViewsCard({ title, body }: NoViewsCardProps) {
  return (
    <div
      className="rounded-2xl border border-border bg-primary-soft p-6"
      data-testid="stats-empty-card"
      role="status"
    >
      <p className="font-serif text-[20px] font-medium tracking-[-0.01em] text-foreground">
        {title}
      </p>
      <p className="mt-2 max-w-xl text-sm text-foreground/80">{body}</p>
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

// seriesForKey extracts the requested numeric field (unique or count)
// from every bucket and returns it as a {date, count} series shape so
// the underlying LineChart — which is agnostic of stats semantics —
// can render either dimension. Falls back to `count` when `unique` is
// missing (legacy cached responses).
function seriesForKey(
  series: StatsTimeBucket[],
  key: "unique" | "count",
): { date: string; count: number }[] {
  return series.map((b) => ({
    date: b.date,
    count: pickValue(b, key),
  }))
}

function pickValue(bucket: StatsTimeBucket, key: "unique" | "count"): number {
  if (key === "unique") {
    return typeof bucket.unique === "number" ? bucket.unique : bucket.count
  }
  return bucket.count
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
