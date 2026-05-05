"use client"

import { useState } from "react"
import { CalendarDays, ChevronDown, ChevronUp } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn, formatCurrency, formatDate } from "@/shared/lib/utils"
import { useCurrentMonth } from "@/shared/hooks/billing-profile/use-current-month"
import type { CurrentMonthLine } from "@/shared/types/billing-profile"

import { Button } from "@/shared/components/ui/button"

/**
 * Compact card showing the running fee total for the current
 * billing month. Sits above the wallet's withdraw block so
 * providers always know how much commission is being accrued.
 *
 * W-19 — Soleil v2 visual port. Card uses the surface/border tokens,
 * the running total is rendered in Geist Mono (`font-mono`), and the
 * icon plate is corail-soft (`bg-primary-soft`). Wallet renders the
 * same component, so its visual stays consistent across both screens.
 */
export function CurrentMonthAggregate() {
  const t = useTranslations("invoicesList")
  const { data, isLoading, isError } = useCurrentMonth()
  const [expanded, setExpanded] = useState(false)

  if (isLoading) {
    return (
      <div
        className="h-28 animate-shimmer rounded-2xl border border-border bg-card"
        style={{ boxShadow: "var(--shadow-card)" }}
      />
    )
  }
  if (isError || !data) return null

  const isEmpty = data.milestone_count === 0
  const totalAmount = formatCurrency(data.total_fee_cents / 100)

  return (
    <section
      className="rounded-2xl border border-border bg-card p-6"
      style={{ boxShadow: "var(--shadow-card)" }}
    >
      <header className="flex items-start gap-4">
        <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-2xl bg-primary-soft text-primary">
          <CalendarDays className="h-5 w-5" strokeWidth={1.6} aria-hidden="true" />
        </div>
        <div className="min-w-0 flex-1">
          <p className="font-mono text-[10px] font-bold uppercase tracking-[0.12em] text-primary">
            {t("currentMonthEyebrow")}
          </p>
          <p className="mt-1 text-[13px] text-muted-foreground">
            {t("currentMonthPeriod", {
              start: formatDate(data.period_start),
              end: formatDate(data.period_end),
            })}
          </p>
        </div>
        {!isEmpty && (
          <p className="shrink-0 font-mono text-[22px] font-semibold tracking-tight text-foreground">
            {totalAmount}
          </p>
        )}
      </header>

      {isEmpty ? (
        <p className="mt-4 text-[14px] italic text-muted-foreground">
          {t("currentMonthEmpty")}
        </p>
      ) : (
        <>
          <p className="mt-4 text-[14px] text-foreground">
            <span className="font-medium">
              {t("currentMonthMilestones", { count: data.milestone_count })}
            </span>{" "}
            <span className="text-muted-foreground">
              · {t("currentMonthCommission")}
            </span>
          </p>
          {data.lines.length > 0 && (
            <Button
              variant="ghost"
              size="auto"
              type="button"
              onClick={() => setExpanded((p) => !p)}
              className={cn(
                "mt-4 inline-flex items-center gap-1.5 rounded-full border border-border bg-card px-3.5 py-1.5",
                "text-[12.5px] font-semibold text-foreground transition-colors hover:border-border-strong",
              )}
              aria-expanded={expanded}
            >
              {expanded
                ? t("currentMonthHideDetail")
                : t("currentMonthShowDetail")}
              {expanded ? (
                <ChevronUp className="h-3.5 w-3.5" aria-hidden="true" />
              ) : (
                <ChevronDown className="h-3.5 w-3.5" aria-hidden="true" />
              )}
            </Button>
          )}
          {expanded && (
            <ul
              role="list"
              className="mt-4 divide-y divide-border rounded-2xl border border-border bg-background"
            >
              {data.lines.map((line) => (
                <li key={line.payment_record_id}>
                  <LineRow line={line} t={t} />
                </li>
              ))}
            </ul>
          )}
        </>
      )}
    </section>
  )
}

type Translator = ReturnType<typeof useTranslations>

function LineRow({ line, t }: { line: CurrentMonthLine; t: Translator }) {
  return (
    <div className="flex items-center justify-between gap-3 px-4 py-3 text-[12.5px]">
      <div className="min-w-0">
        <p className="text-foreground">
          {t("currentMonthLineDelivered", {
            date: formatDate(line.released_at),
          })}
        </p>
        <p className="text-muted-foreground">
          {t("currentMonthLineProposalAmount", {
            amount: formatCurrency(line.proposal_amount_cents / 100),
          })}
        </p>
      </div>
      <p className="shrink-0 font-mono text-[13px] font-semibold text-foreground">
        {formatCurrency(line.platform_fee_cents / 100)}
      </p>
    </div>
  )
}
