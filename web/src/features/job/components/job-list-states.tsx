"use client"

import { Briefcase, Plus } from "lucide-react"
import { useTranslations } from "next-intl"
import { Link } from "@i18n/navigation"
import { cn } from "@/shared/lib/utils"

// W-06 — Empty + skeleton states for the Mes annonces (entreprise)
// listing. Both extracted from the orchestrator to keep each file
// under the 600-line cap.

interface EmptyStateProps {
  canCreate: boolean
}

export function JobListEmptyState({ canCreate }: EmptyStateProps) {
  const t = useTranslations("job")
  return (
    <section
      className="rounded-2xl border border-border bg-card p-10 text-center sm:p-14"
      style={{ boxShadow: "var(--shadow-card)" }}
    >
      <div
        aria-hidden="true"
        className="mx-auto flex h-16 w-16 items-center justify-center rounded-2xl bg-primary-soft text-primary"
      >
        <Briefcase className="h-7 w-7" strokeWidth={1.5} />
      </div>
      <h2 className="mt-5 font-serif text-[22px] font-medium leading-snug tracking-[-0.015em] text-foreground sm:text-[24px]">
        {t("emptyTitle")}
      </h2>
      <p className="mx-auto mt-2 max-w-md text-[14px] leading-relaxed text-muted-foreground">
        {t("emptyBody")}
      </p>
      {canCreate && (
        <Link
          href="/jobs/create"
          className={cn(
            "mt-6 inline-flex items-center justify-center gap-2 rounded-full",
            "px-5 py-2.5 text-[13.5px] font-bold text-primary-foreground",
            "bg-primary transition-all duration-200 ease-out",
            "hover:bg-primary-deep hover:shadow-[0_4px_14px_rgba(232,93,74,0.28)]",
            "active:scale-[0.98]",
            "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background",
          )}
        >
          <Plus className="h-4 w-4" strokeWidth={2} />
          {t("emptyCta")}
        </Link>
      )}
    </section>
  )
}

export function JobListSkeleton() {
  return (
    <div className="space-y-4">
      {Array.from({ length: 3 }).map((_, i) => (
        <div
          key={i}
          className="rounded-2xl border border-border bg-card p-5 sm:p-6"
          style={{ boxShadow: "var(--shadow-card)" }}
        >
          <div className="flex items-start justify-between gap-3">
            <div className="flex-1 space-y-2">
              <div className="h-5 w-32 animate-shimmer rounded-full bg-border" />
              <div className="h-6 w-3/4 animate-shimmer rounded-lg bg-border" />
              <div className="h-4 w-full animate-shimmer rounded-lg bg-border" />
            </div>
            <div className="h-9 w-9 animate-shimmer rounded-full bg-border" />
          </div>
          <div className="mt-5 flex gap-2 border-t border-dashed border-border pt-4">
            <div className="h-6 w-32 animate-shimmer rounded-full bg-border" />
            <div className="h-6 w-24 animate-shimmer rounded-full bg-border" />
          </div>
        </div>
      ))}
    </div>
  )
}
