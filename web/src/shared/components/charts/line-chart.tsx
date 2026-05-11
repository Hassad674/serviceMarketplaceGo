"use client"

import { useId, useMemo } from "react"
import { useFormatter } from "next-intl"
import { cn } from "@/shared/lib/utils"

// LineChart is the full-size sibling of <Sparkline>. Used on the
// /stats deep-dive page — same SVG-only approach, no chart lib. Adds
// axis labels (start / midpoint / end of the period), a discreet
// horizontal baseline grid, and rounded value markers on the last
// data point. Container is responsive (100% width) so the parent
// can lay multiple charts side-by-side without remounting.
//
// D3 extension — optional `secondarySeries` renders a second polyline
// (dashed, no area fill) on top of the primary one. Used by the stats
// page to overlay total views on unique-viewer counts. The two series
// share the same Y axis; the chart scales to whichever maximum is
// larger so neither line clips.

interface ChartSeriesPoint {
  date: string
  count: number
}

interface LineChartProps {
  series: ChartSeriesPoint[]
  /** Aspect-ratio height as a percentage of width (default 30 → 10:3). */
  aspectRatioPct?: number
  /** Accessible title announced to screen readers. */
  title: string
  /** Optional description below the chart for the empty/sparse state. */
  emptyMessage?: string
  /** Tone class applied to the primary line + area gradient (text-primary, etc.). */
  className?: string
  /** Hide axis labels — useful for a cards row where space is tight. */
  hideAxis?: boolean
  /** Optional second polyline drawn on top (dashed, no fill). */
  secondarySeries?: ChartSeriesPoint[]
  /** Legend label for the primary series (shown when secondarySeries is provided). */
  primaryLabel?: string
  /** Legend label for the secondary series. */
  secondaryLabel?: string
}

const VIEWBOX_WIDTH = 600
const VIEWBOX_PADDING_X = 16
const VIEWBOX_PADDING_Y = 24
const VIEWBOX_HEIGHT_DEFAULT = 200

export function LineChart({
  series,
  aspectRatioPct = 30,
  title,
  emptyMessage,
  className,
  hideAxis = false,
  secondarySeries,
  primaryLabel,
  secondaryLabel,
}: LineChartProps) {
  const gradientId = useId()
  const format = useFormatter()

  const combinedMax = useMemo(() => {
    const primary = series.map((p) => p.count)
    const secondary = secondarySeries?.map((p) => p.count) ?? []
    return Math.max(0, ...primary, ...secondary)
  }, [series, secondarySeries])

  const hasAnyData =
    series.some((p) => p.count > 0) ||
    (secondarySeries?.some((p) => p.count > 0) ?? false)

  if (series.length === 0 || !hasAnyData) {
    return (
      <EmptyChart
        title={title}
        message={emptyMessage}
        aspectRatioPct={aspectRatioPct}
      />
    )
  }

  const height = VIEWBOX_HEIGHT_DEFAULT
  const innerWidth = VIEWBOX_WIDTH - VIEWBOX_PADDING_X * 2
  const innerHeight = height - VIEWBOX_PADDING_Y * 2
  const span = Math.max(combinedMax, 1)
  const stepX =
    series.length > 1 ? innerWidth / (series.length - 1) : 0

  const primaryPoints = series.map((p, i) => {
    const x = VIEWBOX_PADDING_X + i * stepX
    const y =
      VIEWBOX_PADDING_Y + innerHeight - (p.count / span) * innerHeight
    return { x, y, value: p.count }
  })

  const linePath =
    primaryPoints.length === 1
      ? `M ${primaryPoints[0].x.toFixed(2)},${primaryPoints[0].y.toFixed(2)} h 0`
      : `M ${primaryPoints.map((p) => `${p.x.toFixed(2)},${p.y.toFixed(2)}`).join(" L ")}`
  const areaPath = `${linePath} L ${(VIEWBOX_PADDING_X + innerWidth).toFixed(2)},${VIEWBOX_PADDING_Y + innerHeight} L ${VIEWBOX_PADDING_X.toFixed(2)},${VIEWBOX_PADDING_Y + innerHeight} Z`

  const secondaryPath = renderSecondaryPath({
    secondary: secondarySeries,
    span,
    stepX,
    innerHeight,
  })

  const lastPoint = primaryPoints[primaryPoints.length - 1]
  const startLabel = formatAxisDate(series[0]?.date, format)
  const endLabel = formatAxisDate(series[series.length - 1]?.date, format)
  const midIdx = Math.floor(series.length / 2)
  const midLabel =
    series.length > 2 ? formatAxisDate(series[midIdx]?.date, format) : null

  return (
    <div
      className={cn(
        "relative w-full overflow-hidden rounded-2xl border border-border bg-card p-5",
        className,
      )}
      role="img"
      aria-label={title}
    >
      <div className="mb-3 flex items-center justify-between gap-3">
        <p className="text-[13px] font-medium text-muted-foreground">{title}</p>
        {primaryLabel && secondaryLabel ? (
          <ChartLegend
            primaryLabel={primaryLabel}
            secondaryLabel={secondaryLabel}
          />
        ) : null}
      </div>
      <svg
        viewBox={`0 0 ${VIEWBOX_WIDTH} ${height}`}
        preserveAspectRatio="none"
        className="block w-full text-primary"
        style={{
          aspectRatio: `${VIEWBOX_WIDTH} / ${(VIEWBOX_WIDTH * aspectRatioPct) / 100}`,
        }}
      >
        <defs>
          <linearGradient id={gradientId} x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stopColor="currentColor" stopOpacity={0.22} />
            <stop offset="100%" stopColor="currentColor" stopOpacity={0} />
          </linearGradient>
        </defs>
        <line
          x1={VIEWBOX_PADDING_X}
          x2={VIEWBOX_WIDTH - VIEWBOX_PADDING_X}
          y1={VIEWBOX_PADDING_Y + innerHeight}
          y2={VIEWBOX_PADDING_Y + innerHeight}
          stroke="currentColor"
          strokeOpacity={0.18}
          strokeWidth={1}
        />
        <path d={areaPath} fill={`url(#${gradientId})`} stroke="none" />
        <path
          d={linePath}
          fill="none"
          stroke="currentColor"
          strokeWidth={2}
          strokeLinecap="round"
          strokeLinejoin="round"
          data-testid="line-chart-primary"
        />
        {secondaryPath ? (
          <path
            d={secondaryPath}
            fill="none"
            stroke="currentColor"
            strokeOpacity={0.55}
            strokeWidth={1.5}
            strokeDasharray="4 4"
            strokeLinecap="round"
            strokeLinejoin="round"
            data-testid="line-chart-secondary"
          />
        ) : null}
        {lastPoint ? (
          <circle cx={lastPoint.x} cy={lastPoint.y} r={4} fill="currentColor" />
        ) : null}
      </svg>
      {!hideAxis ? (
        <div className="mt-3 flex justify-between text-[11px] font-mono uppercase tracking-wider text-muted-foreground">
          <span>{startLabel}</span>
          {midLabel ? <span>{midLabel}</span> : null}
          <span>{endLabel}</span>
        </div>
      ) : null}
    </div>
  )
}

interface ChartLegendProps {
  primaryLabel: string
  secondaryLabel: string
}

// ChartLegend renders the two-line key (corail solid + dashed) next to
// the chart title. Kept inline rather than extracted to shared/ — only
// one consumer today (the stats page); rule of three not yet reached.
function ChartLegend({ primaryLabel, secondaryLabel }: ChartLegendProps) {
  return (
    <div className="flex items-center gap-3 text-[11px] font-mono uppercase tracking-wider text-muted-foreground">
      <span className="flex items-center gap-1.5">
        <span
          aria-hidden
          className="inline-block h-[2px] w-4 rounded-full bg-current text-primary"
        />
        {primaryLabel}
      </span>
      <span className="flex items-center gap-1.5">
        <span
          aria-hidden
          className="inline-block h-[2px] w-4 rounded-full text-primary"
          style={{
            backgroundImage:
              "repeating-linear-gradient(to right, currentColor 0, currentColor 3px, transparent 3px, transparent 6px)",
            opacity: 0.55,
          }}
        />
        {secondaryLabel}
      </span>
    </div>
  )
}

interface SecondaryPathInput {
  secondary: ChartSeriesPoint[] | undefined
  span: number
  stepX: number
  innerHeight: number
}

// renderSecondaryPath projects the secondary series onto the same X
// axis as the primary series. Returns null when no secondary series
// was supplied. The function exists at module scope so unit tests can
// import and exercise it without rendering the chart.
function renderSecondaryPath({
  secondary,
  span,
  stepX,
  innerHeight,
}: SecondaryPathInput): string | null {
  if (!secondary || secondary.length === 0) return null
  const pts = secondary.map((p, i) => {
    const x = VIEWBOX_PADDING_X + i * stepX
    const y =
      VIEWBOX_PADDING_Y + innerHeight - (p.count / span) * innerHeight
    return `${x.toFixed(2)},${y.toFixed(2)}`
  })
  if (pts.length === 1) return `M ${pts[0]} h 0`
  return `M ${pts.join(" L ")}`
}

function formatAxisDate(
  iso: string | undefined,
  format: ReturnType<typeof useFormatter>,
): string {
  if (!iso) return ""
  try {
    return format.dateTime(new Date(iso), { day: "2-digit", month: "short" })
  } catch {
    return ""
  }
}

function EmptyChart({
  title,
  message,
  aspectRatioPct,
}: {
  title: string
  message: string | undefined
  aspectRatioPct: number
}) {
  return (
    <div
      className="rounded-2xl border border-dashed border-border bg-card p-5"
      data-testid="line-chart-empty"
    >
      <p className="text-[13px] font-medium text-muted-foreground">{title}</p>
      <div
        className="mt-3 flex items-center justify-center"
        style={{
          aspectRatio: `${VIEWBOX_WIDTH} / ${(VIEWBOX_WIDTH * aspectRatioPct) / 100}`,
        }}
      >
        <p className="max-w-md text-center text-sm text-muted-foreground">
          {message}
        </p>
      </div>
    </div>
  )
}
