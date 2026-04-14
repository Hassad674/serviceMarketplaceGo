"use client"

import { Clock, Euro, FileText } from "lucide-react"
import { useFormatter, useTranslations } from "next-intl"
import { ReviewCard } from "@/shared/components/ui/review-card"
import {
  useProjectHistory,
  type ProjectHistoryEntry,
} from "@/shared/hooks/profile/use-project-history"

interface ProjectHistorySectionProps {
  orgId: string | undefined
  readOnly?: boolean
  // When true the section skips the query and always renders the
  // empty state. Used by the referrer profile where the freelance
  // project history would otherwise leak across personas (both share
  // the same organization_id).
  forceEmpty?: boolean
  emptyOverride?: {
    title: string
    description: string
  }
}

// ProjectHistorySection is the generic "completed projects" block
// used on every profile persona. Features pass `emptyOverride` when
// they need a persona-specific empty state copy — the referrer page
// uses this to say "no referral deals yet" rather than the default
// freelance phrasing. Keeps data fetching + rendering in one place
// so features don't re-implement the skeleton/empty/error trio.
export function ProjectHistorySection(props: ProjectHistorySectionProps) {
  const { orgId, readOnly = false, forceEmpty = false, emptyOverride } = props
  const t = useTranslations("profile")
  const query = useProjectHistory(forceEmpty ? undefined : orgId)
  const data = forceEmpty ? undefined : query.data
  const isLoading = forceEmpty ? false : query.isLoading
  const isError = forceEmpty ? false : query.isError

  const entries = forceEmpty ? [] : (data?.data ?? [])
  const count = entries.length

  // Public viewers normally want the section hidden when there is
  // nothing to show — except when the consumer ships a persona-
  // specific empty state copy (e.g. the referrer "no deals yet" line),
  // in which case we keep the card visible so visitors see the
  // persona's value prop even with zero projects.
  const hasEmptyOverride = Boolean(emptyOverride)
  if (
    readOnly &&
    !isLoading &&
    !hasEmptyOverride &&
    (isError || count === 0)
  ) {
    return null
  }

  return (
    <section className="bg-card border border-border rounded-xl p-4 shadow-sm sm:p-6">
      <HistoryHeader count={count} />

      {isLoading ? (
        <HistorySkeleton />
      ) : isError ? (
        <div className="rounded-xl border border-destructive/20 bg-destructive/5 p-4 text-sm text-destructive">
          {t("historyLoadError")}
        </div>
      ) : count === 0 ? (
        <EmptyState override={emptyOverride} />
      ) : (
        <ul className="space-y-4">
          {entries.map((entry) => (
            <li key={entry.proposal_id}>
              <ProjectHistoryEntryCard entry={entry} />
            </li>
          ))}
        </ul>
      )}
    </section>
  )
}

interface HistoryHeaderProps {
  count: number
}

function HistoryHeader({ count }: HistoryHeaderProps) {
  const t = useTranslations("profile")
  return (
    <div className="flex items-center gap-3 mb-4">
      <h2 className="text-base font-semibold text-foreground sm:text-lg">
        {t("projectHistory")}
      </h2>
      {count > 0 ? (
        <span className="rounded-full bg-muted text-muted-foreground px-2.5 py-0.5 text-xs font-medium">
          {t("completedCount", { count })}
        </span>
      ) : null}
    </div>
  )
}

interface EmptyStateProps {
  override: ProjectHistorySectionProps["emptyOverride"]
}

function EmptyState({ override }: EmptyStateProps) {
  const t = useTranslations("profile")
  const title = override?.title ?? t("noProjects")
  const description = override?.description ?? t("projectsAppearHere")
  return (
    <div className="flex flex-col items-center justify-center py-10 text-center">
      <div className="w-12 h-12 rounded-full bg-muted flex items-center justify-center mb-3">
        <FileText
          className="w-6 h-6 text-muted-foreground"
          aria-hidden="true"
        />
      </div>
      <p className="text-base font-medium text-foreground mb-1">{title}</p>
      <p className="text-sm text-muted-foreground italic">{description}</p>
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

interface EntryCardProps {
  entry: ProjectHistoryEntry
}

function ProjectHistoryEntryCard({ entry }: EntryCardProps) {
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
