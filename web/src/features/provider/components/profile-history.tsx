"use client"

import { FileText } from "lucide-react"
import { useTranslations } from "next-intl"
import { useProjectHistory } from "../hooks/use-project-history"
import { ProjectHistoryCard } from "./project-history-card"

interface ProfileHistoryProps {
  userId: string | undefined
  readOnly?: boolean
}

/**
 * Project history section — unified view of completed missions with their
 * (optional) reviews. Replaces the previous standalone placeholder AND the
 * separate reviews section.
 */
export function ProfileHistory({ userId, readOnly = false }: ProfileHistoryProps) {
  const t = useTranslations("profile")
  const { data, isLoading, isError } = useProjectHistory(userId)

  const entries = data?.data ?? []
  const count = entries.length

  // Public profile: hide the entire section when there's nothing to show
  // (empty result or error) — a visitor doesn't care about a load failure.
  // On the private profile (readOnly === false) we always render so the
  // owner sees the empty/error state and can act on it.
  if (readOnly && !isLoading && (isError || count === 0)) return null

  return (
    <section className="bg-card border border-border rounded-xl p-4 shadow-sm sm:p-6">
      {/* Header */}
      <div className="flex items-center gap-3 mb-4">
        <h2 className="text-base font-semibold text-foreground sm:text-lg">
          {t("projectHistory")}
        </h2>
        {count > 0 && (
          <span className="rounded-full bg-muted text-muted-foreground px-2.5 py-0.5 text-xs font-medium">
            {t("completedCount", { count })}
          </span>
        )}
      </div>

      {/* Content */}
      {isLoading ? (
        <ProjectHistorySkeleton />
      ) : isError ? (
        <div className="rounded-xl border border-destructive/20 bg-destructive/5 p-4 text-sm text-destructive">
          {t("historyLoadError")}
        </div>
      ) : count === 0 ? (
        <EmptyState />
      ) : (
        <div className="space-y-4">
          {entries.map((entry) => (
            <ProjectHistoryCard key={entry.proposal_id} entry={entry} />
          ))}
        </div>
      )}
    </section>
  )
}

function EmptyState() {
  const t = useTranslations("profile")
  return (
    <div className="flex flex-col items-center justify-center py-8 text-center">
      <div className="w-12 h-12 rounded-full bg-muted flex items-center justify-center mb-3">
        <FileText
          className="w-5 h-5 text-muted-foreground"
          aria-hidden="true"
        />
      </div>
      <p className="text-sm font-medium text-foreground mb-1">
        {t("noProjects")}
      </p>
      <p className="text-sm text-muted-foreground italic">
        {t("projectsAppearHere")}
      </p>
    </div>
  )
}

function ProjectHistorySkeleton() {
  return (
    <div className="space-y-4">
      {[0, 1, 2].map((i) => (
        <div
          key={i}
          className="animate-shimmer h-32 rounded-2xl bg-gradient-to-br from-muted via-muted/60 to-muted"
        />
      ))}
    </div>
  )
}
