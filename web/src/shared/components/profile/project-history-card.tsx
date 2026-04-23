"use client"

import { Clock, Euro } from "lucide-react"
import { useFormatter, useTranslations } from "next-intl"
import { ReviewCard } from "@/shared/components/ui/review-card"
import type { Review } from "@/shared/types/review"

// CardEntry is the minimal counterparty-agnostic shape the card
// needs. Both the provider-side project history (proposals where the
// org delivered) and the client-side project history (proposals the
// org paid for) flatten into this same shape — the only difference
// lives upstream (which endpoint populates the list), so the card is
// deliberately persona-neutral.
export type ProjectHistoryCardEntry = {
  proposal_id: string
  title: string
  amount: number
  currency?: string
  completed_at: string
  review: Review | null
}

interface ProjectHistoryCardProps {
  entry: ProjectHistoryCardEntry
}

// ProjectHistoryCard renders one completed project row: amount pill
// + completion date in the header, title when set, and either the
// embedded review or the "awaiting review" placeholder below. Every
// profile surface (provider, freelance, referrer, client) uses this
// same card so a review that shows up on one page looks identical on
// every other page where the same proposal appears.
export function ProjectHistoryCard({ entry }: ProjectHistoryCardProps) {
  const t = useTranslations("profile")
  const format = useFormatter()
  const amount = entry.amount / 100
  const completedDate = new Date(entry.completed_at)
  const showTitle = entry.title.trim().length > 0

  return (
    <article className="rounded-2xl border border-border bg-card p-5 shadow-sm transition-colors hover:border-rose-200">
      <header className="flex flex-wrap items-center justify-between gap-3">
        <div className="inline-flex items-center gap-1.5 rounded-full bg-gradient-to-r from-rose-50 to-rose-100/60 px-3 py-1.5 text-sm font-semibold text-rose-700">
          <Euro className="h-3.5 w-3.5" strokeWidth={2.5} />
          {format.number(amount, {
            style: "currency",
            currency: entry.currency || "EUR",
            maximumFractionDigits: 0,
          })}
        </div>
        <div className="inline-flex items-center gap-1.5 text-xs text-muted-foreground">
          <Clock className="h-3.5 w-3.5" />
          <time dateTime={entry.completed_at}>
            {t("completedOn", {
              date: format.dateTime(completedDate, {
                year: "numeric",
                month: "short",
                day: "numeric",
              }),
            })}
          </time>
        </div>
      </header>

      {showTitle ? (
        <h3 className="mt-3 text-base font-semibold text-foreground">
          {entry.title}
        </h3>
      ) : null}

      <div className="mt-4">
        {entry.review ? (
          <ReviewCard review={entry.review} />
        ) : (
          <div className="flex items-center gap-2 rounded-xl border border-dashed border-border bg-muted/30 p-4 text-sm text-muted-foreground">
            <Clock className="h-4 w-4 shrink-0" />
            <span>{t("awaitingReview")}</span>
          </div>
        )}
      </div>
    </article>
  )
}
