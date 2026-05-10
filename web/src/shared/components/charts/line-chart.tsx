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

interface LineChartProps {
  series: { date: string; count: number }[]
  /** Aspect-ratio height as a percentage of width (default 30 → 10:3). */
  aspectRatioPct?: number
  /** Accessible title announced to screen readers. */
  title: string
  /** Optional description below the chart for the empty/sparse state. */
  emptyMessage?: string
  /** Tone class applied to the line + area gradient (text-primary, etc.). */
  className?: string
  /** Hide axis labels — useful for a cards row where space is tight. */
  hideAxis?: boolean
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
}: LineChartProps) {
  const gradientId = useId()
  const format = useFormatter()

  const computed = useMemo(() => computeChart(series), [series])

  if (computed.values.length === 0) {
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
  const stepX =
    computed.values.length > 1 ? innerWidth / (computed.values.length - 1) : 0

  const points = computed.values.map((value, i) => {
    const x = VIEWBOX_PADDING_X + i * stepX
    const y =
      VIEWBOX_PADDING_Y +
      innerHeight -
      ((value - computed.min) / Math.max(computed.span, 1)) * innerHeight
    return { x, y, value }
  })

  const linePath = points.length === 1
    ? `M ${points[0].x.toFixed(2)},${points[0].y.toFixed(2)} h 0`
    : `M ${points.map((p) => `${p.x.toFixed(2)},${p.y.toFixed(2)}`).join(" L ")}`
  const areaPath = `${linePath} L ${(VIEWBOX_PADDING_X + innerWidth).toFixed(2)},${VIEWBOX_PADDING_Y + innerHeight} L ${VIEWBOX_PADDING_X.toFixed(2)},${VIEWBOX_PADDING_Y + innerHeight} Z`

  const lastPoint = points[points.length - 1]
  const startLabel = formatAxisDate(series[0]?.date, format)
  const endLabel = formatAxisDate(series[series.length - 1]?.date, format)

  return (
    <div
      className={cn(
        "relative w-full overflow-hidden rounded-2xl border border-border bg-card p-5",
        className,
      )}
      role="img"
      aria-label={title}
    >
      <p className="mb-3 text-[13px] font-medium text-muted-foreground">{title}</p>
      <svg
        viewBox={`0 0 ${VIEWBOX_WIDTH} ${height}`}
        preserveAspectRatio="none"
        className="block w-full text-primary"
        style={{ aspectRatio: `${VIEWBOX_WIDTH} / ${(VIEWBOX_WIDTH * aspectRatioPct) / 100}` }}
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
        />
        {lastPoint ? (
          <circle
            cx={lastPoint.x}
            cy={lastPoint.y}
            r={4}
            fill="currentColor"
          />
        ) : null}
      </svg>
      {!hideAxis ? (
        <div className="mt-3 flex justify-between text-[11px] font-mono uppercase tracking-wider text-muted-foreground">
          <span>{startLabel}</span>
          <span>{endLabel}</span>
        </div>
      ) : null}
    </div>
  )
}

interface ChartGeometry {
  values: number[]
  min: number
  max: number
  span: number
}

function computeChart(series: { count: number }[]): ChartGeometry {
  const values = series.map((p) => p.count)
  if (values.length === 0) {
    return { values: [], min: 0, max: 0, span: 1 }
  }
  const max = Math.max(...values, 0)
  const min = Math.min(...values, 0)
  const span = Math.max(max - min, 1)
  return { values, min, max, span }
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
    <div className="rounded-2xl border border-dashed border-border bg-card p-5">
      <p className="text-[13px] font-medium text-muted-foreground">{title}</p>
      <div
        className="mt-3 flex items-center justify-center"
        style={{ aspectRatio: `${VIEWBOX_WIDTH} / ${(VIEWBOX_WIDTH * aspectRatioPct) / 100}` }}
      >
        <p className="max-w-md text-center text-sm text-muted-foreground">
          {message}
        </p>
      </div>
    </div>
  )
}
