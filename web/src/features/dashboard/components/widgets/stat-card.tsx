"use client"

import type { LucideIcon } from "lucide-react"
import { ArrowDown, ArrowUp, Minus } from "lucide-react"
import { cn } from "@/shared/lib/utils"
import { Sparkline } from "@/shared/components/charts/sparkline"

// StatCard is the shared "big number + sparkline" tile used across the
// dashboard's top-of-page metric strip. Every dashboard layout
// (Provider/Agency, Enterprise, Referrer) renders four of these in a
// responsive grid. The variant accents allow the cards to cycle
// through the Soleil v2 tone palette without leaking colour decisions
// into the layout level.

interface StatCardProps {
  label: string
  /** Pre-formatted value (number, currency, position, etc.). */
  value: string
  /** Optional sub-label rendered below the big number. */
  subLabel?: string
  /** Sparkline data — last N daily counts. Empty array hides the chart. */
  series?: number[]
  /** Lucide icon component rendered in the corner badge. */
  icon: LucideIcon
  /** Soleil v2 tone variant — controls icon backdrop and sparkline colour. */
  tone?: "primary" | "success" | "amber" | "deep"
  /**
   * Delta-vs-previous-period in percent. Positive → up arrow, negative
   * → down arrow, zero / undefined → neutral pill. Hidden when
   * `hideDelta` is true (e.g. cards where direction is meaningless).
   */
  deltaPct?: number
  hideDelta?: boolean
  /** Optional state applied to skeleton-aware consumers. */
  isLoading?: boolean
}

const TONE_CLASS: Record<NonNullable<StatCardProps["tone"]>, { bg: string; icon: string; line: string }> = {
  primary: {
    bg: "bg-primary-soft",
    icon: "text-primary",
    line: "text-primary",
  },
  success: {
    bg: "bg-success-soft",
    icon: "text-success",
    line: "text-success",
  },
  amber: {
    bg: "bg-amber-soft",
    icon: "text-foreground",
    line: "text-foreground",
  },
  deep: {
    bg: "bg-pink-soft",
    icon: "text-primary-deep",
    line: "text-primary-deep",
  },
}

export function StatCard(props: StatCardProps) {
  const tone = TONE_CLASS[props.tone ?? "primary"]
  const Icon = props.icon

  return (
    <div
      className={cn(
        "group rounded-2xl border border-border bg-card p-5 shadow-card",
        "transition-all duration-200 hover:-translate-y-0.5 hover:border-border-strong",
      )}
    >
      <div className="flex items-start justify-between">
        <div
          className={cn(
            "flex h-11 w-11 items-center justify-center rounded-2xl",
            tone.bg,
          )}
        >
          <Icon className={cn("h-5 w-5", tone.icon)} strokeWidth={1.5} aria-hidden />
        </div>
        {props.series && props.series.length >= 2 ? (
          <Sparkline values={props.series} className={tone.line} ariaLabel={props.label} />
        ) : null}
      </div>
      <p className="mt-4 font-mono text-[11px] font-bold uppercase tracking-[0.12em] text-muted-foreground">
        {props.label}
      </p>
      <p
        className={cn(
          "mt-1 font-serif text-[30px] font-medium leading-tight tracking-[-0.02em] text-foreground",
          props.isLoading && "animate-pulse text-muted-foreground/40",
        )}
      >
        {props.isLoading ? "—" : props.value}
      </p>
      <DeltaPill
        deltaPct={props.deltaPct}
        subLabel={props.subLabel}
        hideDelta={props.hideDelta}
      />
    </div>
  )
}

interface DeltaPillProps {
  deltaPct: number | undefined
  subLabel: string | undefined
  hideDelta: boolean | undefined
}

function DeltaPill({ deltaPct, subLabel, hideDelta }: DeltaPillProps) {
  if (hideDelta || typeof deltaPct !== "number") {
    if (subLabel) {
      return (
        <p className="mt-1.5 text-[12px] text-muted-foreground">{subLabel}</p>
      )
    }
    return null
  }
  const positive = deltaPct > 0
  const negative = deltaPct < 0
  const Icon = positive ? ArrowUp : negative ? ArrowDown : Minus
  const tone = positive
    ? "text-success bg-success-soft"
    : negative
      ? "text-primary bg-primary-soft"
      : "text-muted-foreground bg-muted"
  return (
    <div className="mt-1.5 flex items-center gap-1.5">
      <span
        className={cn(
          "inline-flex items-center gap-0.5 rounded-full px-2 py-0.5 text-[11px] font-semibold",
          tone,
        )}
      >
        <Icon className="h-3 w-3" strokeWidth={2} aria-hidden />
        {Math.abs(deltaPct).toFixed(0)}%
      </span>
      {subLabel ? (
        <span className="text-[11px] text-muted-foreground">{subLabel}</span>
      ) : null}
    </div>
  )
}
