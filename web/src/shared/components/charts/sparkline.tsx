"use client"

import { useId } from "react"
import { cn } from "@/shared/lib/utils"

// Sparkline is a tiny inline chart that lives inside a stat card.
// Pure SVG, no chart library dependency — keeps the bundle below the
// 200 KB budget on the dashboard route. Renders an empty axis when
// the series is too short to plot a meaningful trend (< 2 points)
// instead of throwing or drawing a flat line.

interface SparklineProps {
  values: number[]
  /** Width in CSS pixels — defaults to 96 (matches the card layout). */
  width?: number
  /** Height in CSS pixels — defaults to 32. */
  height?: number
  /** Line / fill stroke colour — defaults to currentColor (parent text). */
  className?: string
  /** Accessible label announced to screen readers. */
  ariaLabel?: string
}

export function Sparkline({
  values,
  width = 96,
  height = 32,
  className,
  ariaLabel,
}: SparklineProps) {
  const gradientId = useId()

  if (values.length < 2) {
    return (
      <svg
        width={width}
        height={height}
        viewBox={`0 0 ${width} ${height}`}
        className={cn("text-muted-foreground/40", className)}
        aria-label={ariaLabel}
        role={ariaLabel ? "img" : "presentation"}
      >
        <line
          x1={0}
          x2={width}
          y1={height - 1}
          y2={height - 1}
          stroke="currentColor"
          strokeWidth={1}
          strokeDasharray="2 3"
        />
      </svg>
    )
  }

  const max = Math.max(...values, 1)
  const min = Math.min(...values, 0)
  const span = Math.max(max - min, 1)
  const stepX = width / (values.length - 1)

  const points = values.map((v, i) => {
    const x = i * stepX
    const y = height - ((v - min) / span) * (height - 2) - 1
    return `${x.toFixed(2)},${y.toFixed(2)}`
  })

  const linePath = `M ${points.join(" L ")}`
  const areaPath = `${linePath} L ${(width).toFixed(2)},${height} L 0,${height} Z`

  return (
    <svg
      width={width}
      height={height}
      viewBox={`0 0 ${width} ${height}`}
      className={cn("text-primary", className)}
      aria-label={ariaLabel}
      role={ariaLabel ? "img" : "presentation"}
    >
      <defs>
        <linearGradient id={gradientId} x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stopColor="currentColor" stopOpacity={0.18} />
          <stop offset="100%" stopColor="currentColor" stopOpacity={0} />
        </linearGradient>
      </defs>
      <path d={areaPath} fill={`url(#${gradientId})`} stroke="none" />
      <path
        d={linePath}
        fill="none"
        stroke="currentColor"
        strokeWidth={1.5}
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  )
}
