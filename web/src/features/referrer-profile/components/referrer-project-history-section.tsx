"use client"

import { Clock, FileText, Star } from "lucide-react"
import { useFormatter, useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import { ReviewCard } from "@/shared/components/ui/review-card"
import { useReferrerReputation } from "../hooks/use-referrer-reputation"
import type { ReferrerProjectHistoryEntry } from "../api/reputation-api"

interface ReferrerProjectHistorySectionProps {
  orgId: string | undefined
}

// ReferrerProjectHistorySection is the apporteur-specific reputation
// surface: a dedicated rating (distinct from the user's freelance
// rating) plus a cursor-paginated history of attributed missions.
// Visual grammar mirrors the freelance project-history card so users
// find it familiar, but the labels are unambiguous about scope —
// "Projets apportés" / "Avis des clients sur les prestataires
// recommandés" — and BOTH client and provider identities are
// intentionally absent: the apporteur's recommendation graph stays
// confidential. Every introduced provider is surfaced as the static
// "Prestataire introduit" label so the public profile never leaks
// who the apporteur recommends.
export function ReferrerProjectHistorySection(
  props: ReferrerProjectHistorySectionProps,
) {
  const { orgId } = props
  const t = useTranslations("profile.referrer")
  const query = useReferrerReputation(orgId)

  const firstPage = query.data?.pages[0]
  const entries =
    query.data?.pages.flatMap((page) => page.history) ?? []
  const ratingAvg = firstPage?.rating_avg ?? 0
  const reviewCount = firstPage?.review_count ?? 0

  return (
    <section className="bg-card border border-border rounded-xl p-4 shadow-sm sm:p-6">
      <SectionHeader
        ratingAvg={ratingAvg}
        reviewCount={reviewCount}
      />

      {query.isLoading ? (
        <HistorySkeleton />
      ) : query.isError ? (
        <div className="rounded-xl border border-destructive/20 bg-destructive/5 p-4 text-sm text-destructive">
          {t("reputationLoadError")}
        </div>
      ) : entries.length === 0 ? (
        <EmptyState />
      ) : (
        <>
          <ul className="space-y-4">
            {entries.map((entry) => (
              <li key={entry.proposal_id}>
                <HistoryEntryCard entry={entry} />
              </li>
            ))}
          </ul>
          {query.hasNextPage ? (
            <div className="mt-5 flex justify-center">
              <button
                type="button"
                onClick={() => query.fetchNextPage()}
                disabled={query.isFetchingNextPage}
                className={cn(
                  "inline-flex items-center gap-2 rounded-full border border-border bg-background px-4 py-2 text-sm font-medium text-foreground transition-colors",
                  "hover:border-rose-200 hover:bg-rose-50/50 dark:hover:bg-rose-950/20",
                  "disabled:opacity-60 disabled:cursor-not-allowed",
                )}
              >
                {query.isFetchingNextPage
                  ? t("reputationLoadMoreLoading")
                  : t("reputationLoadMore")}
              </button>
            </div>
          ) : null}
        </>
      )}
    </section>
  )
}

interface SectionHeaderProps {
  ratingAvg: number
  reviewCount: number
}

function SectionHeader({ ratingAvg, reviewCount }: SectionHeaderProps) {
  const t = useTranslations("profile.referrer")
  const format = useFormatter()
  const formattedAvg = format.number(ratingAvg, {
    minimumFractionDigits: 1,
    maximumFractionDigits: 1,
  })

  return (
    <div className="mb-4 flex flex-col gap-1">
      <div className="flex flex-wrap items-baseline justify-between gap-2">
        <h2 className="text-base font-semibold text-foreground sm:text-lg">
          {t("reputationSectionTitle")}
        </h2>
        <span
          className="inline-flex items-center gap-1.5 rounded-full bg-gradient-to-r from-rose-50 to-rose-100/60 px-3 py-1 text-xs font-semibold text-rose-700 dark:from-rose-950/40 dark:to-rose-900/40 dark:text-rose-200"
          aria-label={`${t("reputationRatingLabel")} · ${formattedAvg} / 5`}
        >
          <Star
            className="h-3.5 w-3.5 fill-current"
            strokeWidth={2.5}
            aria-hidden="true"
          />
          {t("reputationRatingLabel")} ·{" "}
          {t("reputationRatingValue", {
            avg: formattedAvg,
            count: reviewCount,
          })}
        </span>
      </div>
      <p className="text-xs text-muted-foreground">
        {t("reputationSectionSubtitle")}
      </p>
    </div>
  )
}

function EmptyState() {
  const t = useTranslations("profile.referrer")
  return (
    <div className="flex flex-col items-center justify-center py-10 text-center">
      <div className="mb-3 flex h-12 w-12 items-center justify-center rounded-full bg-muted">
        <FileText
          className="h-6 w-6 text-muted-foreground"
          aria-hidden="true"
        />
      </div>
      <p className="mb-1 text-base font-medium text-foreground">
        {t("reputationEmptyTitle")}
      </p>
      <p className="text-sm italic text-muted-foreground">
        {t("reputationEmptyDescription")}
      </p>
    </div>
  )
}

function HistorySkeleton() {
  return (
    <div className="space-y-3" role="status" aria-live="polite">
      {[0, 1, 2].map((i) => (
        <div
          key={i}
          className="h-28 rounded-2xl border border-border bg-muted/40 animate-shimmer"
        />
      ))}
    </div>
  )
}

interface HistoryEntryCardProps {
  entry: ReferrerProjectHistoryEntry
}

function HistoryEntryCard({ entry }: HistoryEntryCardProps) {
  const t = useTranslations("profile.referrer")
  const format = useFormatter()
  const hasReview = entry.review !== null && entry.review !== undefined
  const timelineDate = entry.completed_at
    ? new Date(entry.completed_at)
    : new Date(entry.attributed_at)

  return (
    <article
      className={cn(
        "rounded-2xl border bg-card p-5 shadow-sm transition-colors",
        hasReview
          ? "border-border hover:border-rose-200"
          : "border-border/60 bg-muted/20",
      )}
    >
      <header className="flex flex-wrap items-center justify-between gap-3">
        <StatusBadge status={entry.proposal_status} />
        <div className="inline-flex items-center gap-1.5 text-xs text-muted-foreground">
          <Clock className="h-3.5 w-3.5" aria-hidden="true" />
          <time dateTime={timelineDate.toISOString()}>
            {format.dateTime(timelineDate, {
              year: "numeric",
              month: "short",
              day: "numeric",
            })}
          </time>
        </div>
      </header>

      {entry.proposal_title ? (
        <h3 className="mt-3 text-base font-semibold text-foreground">
          {entry.proposal_title}
        </h3>
      ) : null}

      {/*
        Static "Prestataire introduit" label — the apporteur's public
        profile never exposes WHO was introduced (Modèle A
        confidentiality), only the OUTCOME (status + review).
      */}
      <p className="mt-2 text-sm font-medium text-foreground">
        {t("introducedProvider")}
      </p>

      <div className="mt-4">
        {hasReview && entry.review ? (
          // DRY: reuse the shared ReviewCard primitive so the apporteur
          // surface renders stars + sub-criteria + comment + video
          // identically to the freelance project history.
          <ReviewCard review={entry.review} />
        ) : (
          <div className="flex items-center gap-2 rounded-xl border border-dashed border-border bg-muted/30 p-3 text-sm text-muted-foreground">
            <Clock className="h-4 w-4 shrink-0" aria-hidden="true" />
            <span>{t("reputationNoReviewBadge")}</span>
          </div>
        )}
      </div>
    </article>
  )
}

interface StatusBadgeProps {
  status: string
}

function StatusBadge({ status }: StatusBadgeProps) {
  const t = useTranslations("profile.referrer")
  const style = statusStyle(status)
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1 rounded-full px-2.5 py-1 text-xs font-medium",
        style,
      )}
    >
      {statusLabel(status, t)}
    </span>
  )
}

function statusLabel(
  status: string,
  t: ReturnType<typeof useTranslations<"profile.referrer">>,
): string {
  switch (status) {
    case "completed":
      return t("reputationStatusCompleted")
    case "disputed":
      return t("reputationStatusDisputed")
    case "completion_requested":
      return t("reputationStatusCompletionRequested")
    case "active":
      return t("reputationStatusActive")
    case "paid":
      return t("reputationStatusPaid")
    case "accepted":
      return t("reputationStatusAccepted")
    case "pending":
      return t("reputationStatusPending")
    case "declined":
      return t("reputationStatusDeclined")
    case "withdrawn":
      return t("reputationStatusWithdrawn")
    default:
      return t("reputationStatusUnknown")
  }
}

function statusStyle(status: string): string {
  switch (status) {
    case "completed":
      return "bg-emerald-50 text-emerald-700 dark:bg-emerald-950/40 dark:text-emerald-200"
    case "disputed":
      return "bg-destructive/10 text-destructive"
    case "active":
    case "completion_requested":
    case "paid":
    case "accepted":
      return "bg-sky-50 text-sky-700 dark:bg-sky-950/40 dark:text-sky-200"
    case "declined":
    case "withdrawn":
      return "bg-muted text-muted-foreground"
    default:
      return "bg-muted text-muted-foreground"
  }
}
