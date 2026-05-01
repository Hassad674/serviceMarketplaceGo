"use client"

import { useState } from "react"
import { CalendarDays, ChevronDown, ChevronUp } from "lucide-react"
import { cn, formatCurrency, formatDate } from "@/shared/lib/utils"
import { useCurrentMonth } from "../hooks/use-current-month"
import type { CurrentMonthLine } from "../types"

import { Button } from "@/shared/components/ui/button"
/**
 * Compact card showing the running fee total for the current
 * billing month. Sits above the wallet's withdraw block so
 * providers always know how much commission is being accrued.
 */
export function CurrentMonthAggregate() {
  const { data, isLoading, isError } = useCurrentMonth()
  const [expanded, setExpanded] = useState(false)

  if (isLoading) {
    return (
      <div className="h-24 animate-shimmer rounded-2xl bg-slate-100 dark:bg-slate-800" />
    )
  }
  if (isError || !data) return null

  const isEmpty = data.milestone_count === 0

  return (
    <section className="rounded-2xl border border-slate-100 bg-white p-5 shadow-sm dark:border-slate-700 dark:bg-slate-900">
      <header className="flex items-start gap-3">
        <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-rose-50 text-rose-600 dark:bg-rose-500/10 dark:text-rose-300">
          <CalendarDays className="h-4 w-4" aria-hidden="true" />
        </div>
        <div className="min-w-0 flex-1">
          <h2 className="text-sm font-semibold text-slate-900 dark:text-white">
            Mois en cours
          </h2>
          <p className="text-xs text-slate-500 dark:text-slate-400">
            Du {formatDate(data.period_start)} au {formatDate(data.period_end)}
          </p>
        </div>
      </header>

      {isEmpty ? (
        <p className="mt-3 text-sm text-slate-500 dark:text-slate-400">
          Aucun jalon livré ce mois-ci.
        </p>
      ) : (
        <>
          <p className="mt-3 text-sm text-slate-700 dark:text-slate-200">
            <strong className="font-semibold">{data.milestone_count}</strong>{" "}
            {data.milestone_count > 1 ? "jalons livrés" : "jalon livré"} ·{" "}
            <strong className="font-mono font-semibold">
              {formatCurrency(data.total_fee_cents / 100)}
            </strong>{" "}
            de commission
          </p>
          {data.lines.length > 0 && (
            <Button variant="ghost" size="auto"
              type="button"
              onClick={() => setExpanded((p) => !p)}
              className={cn(
                "mt-3 inline-flex items-center gap-1 text-xs font-medium",
                "text-rose-600 hover:underline dark:text-rose-400",
              )}
              aria-expanded={expanded}
            >
              {expanded ? "Masquer le détail" : "Voir le détail"}
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
              className="mt-3 divide-y divide-slate-100 rounded-lg border border-slate-100 dark:divide-slate-800 dark:border-slate-700"
            >
              {data.lines.map((line) => (
                <li key={line.payment_record_id}>
                  <LineRow line={line} />
                </li>
              ))}
            </ul>
          )}
        </>
      )}
    </section>
  )
}

function LineRow({ line }: { line: CurrentMonthLine }) {
  return (
    <div className="flex items-center justify-between gap-3 px-3 py-2 text-xs">
      <div>
        <p className="text-slate-700 dark:text-slate-200">
          Livré le {formatDate(line.released_at)}
        </p>
        <p className="text-slate-500 dark:text-slate-400">
          Sur {formatCurrency(line.proposal_amount_cents / 100)} de prestation
        </p>
      </div>
      <p className="font-mono font-semibold text-slate-900 dark:text-white">
        {formatCurrency(line.platform_fee_cents / 100)}
      </p>
    </div>
  )
}
