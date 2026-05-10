"use client"

import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import type { KeywordStat } from "../api/stats-api"

// KeywordsTable renders the "Top mots-clés" table on the /stats deep
// dive. Each row shows the keyword, the volume, the average position,
// and an inline horizontal bar whose width is proportional to the row
// volume against the table maximum. The bar is purely visual — the
// numeric value next to it is the source of truth for screen readers.

interface KeywordsTableProps {
  rows: KeywordStat[]
  isLoading?: boolean
}

export function KeywordsTable({ rows, isLoading }: KeywordsTableProps) {
  const t = useTranslations("stats.keywords")

  if (isLoading) {
    return (
      <div className="space-y-2" aria-busy="true">
        {Array.from({ length: 4 }).map((_, i) => (
          <div key={i} className="h-10 animate-pulse rounded-xl bg-muted/50" />
        ))}
      </div>
    )
  }
  if (rows.length === 0) {
    return (
      <div className="rounded-2xl border border-dashed border-border bg-card p-6 text-center">
        <p className="text-sm text-muted-foreground">{t("empty")}</p>
      </div>
    )
  }
  const max = Math.max(...rows.map((row) => row.count), 1)
  return (
    <div className="overflow-hidden rounded-2xl border border-border bg-card">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-border bg-muted/30">
            <KeywordHeader>{t("columnKeyword")}</KeywordHeader>
            <KeywordHeader className="text-right">{t("columnVolume")}</KeywordHeader>
            <KeywordHeader className="text-right">{t("columnPosition")}</KeywordHeader>
          </tr>
        </thead>
        <tbody>
          {rows.map((row) => (
            <KeywordRow key={row.keyword} row={row} max={max} />
          ))}
        </tbody>
      </table>
    </div>
  )
}

function KeywordHeader({
  children,
  className,
}: {
  children: React.ReactNode
  className?: string
}) {
  return (
    <th
      scope="col"
      className={cn(
        "px-4 py-2 font-mono text-[11px] font-bold uppercase tracking-[0.12em] text-muted-foreground",
        className,
      )}
    >
      {children}
    </th>
  )
}

function KeywordRow({ row, max }: { row: KeywordStat; max: number }) {
  const widthPct = (row.count / max) * 100
  return (
    <tr className="border-b border-border last:border-b-0">
      <td className="px-4 py-2.5 font-medium text-foreground">{row.keyword}</td>
      <td className="px-4 py-2.5 text-right">
        <KeywordVolume count={row.count} widthPct={widthPct} />
      </td>
      <td className="px-4 py-2.5 text-right font-mono text-foreground">
        {formatPosition(row.avg_position)}
      </td>
    </tr>
  )
}

function KeywordVolume({ count, widthPct }: { count: number; widthPct: number }) {
  // The width of the bar is data-driven (proportional to row volume
  // against table max) — there is no static utility class that maps to
  // a percentage so we use an inline `style`. This is one of two
  // legitimate exceptions to "no inline style" — chart geometry where
  // the value is computed from data, the other being SVG transforms.
  return (
    <span className="inline-flex w-full items-center justify-end gap-2">
      <span
        aria-hidden
        className="h-1.5 w-20 max-w-full overflow-hidden rounded-full bg-muted"
      >
        <span
          className="block h-full rounded-full bg-primary"
          data-testid="keyword-volume-bar"
          style={{ width: `${widthPct}%` }}
        />
      </span>
      <span className="font-mono text-foreground">{count}</span>
    </span>
  )
}

function formatPosition(p: number | null): string {
  if (p === null || Number.isNaN(p)) return "—"
  return p.toFixed(1)
}
