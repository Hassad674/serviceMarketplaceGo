"use client"

import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import { Button } from "@/shared/components/ui/button"

// W-06 — Status filter pills above the job list. Only `open` and
// `closed` are exposed by the data shape; the JSX source's
// "En pause / Brouillon / Archivée" filters are flagged out of scope.

export type JobStatusFilter = "all" | "open" | "closed"

interface JobListFilterPillsProps {
  filter: JobStatusFilter
  onChange: (next: JobStatusFilter) => void
  counts: { all: number; open: number; closed: number }
}

export function JobListFilterPills({
  filter,
  onChange,
  counts,
}: JobListFilterPillsProps) {
  const t = useTranslations("job")
  const items: Array<{ key: JobStatusFilter; label: string; count: number }> = [
    { key: "all", label: t("filterAll"), count: counts.all },
    { key: "open", label: t("filterOpen"), count: counts.open },
    { key: "closed", label: t("filterClosed"), count: counts.closed },
  ]
  return (
    <div
      className="-mx-1 flex flex-wrap items-center gap-2 px-1"
      role="tablist"
      aria-label={t("title")}
    >
      {items.map((item) => {
        const active = filter === item.key
        return (
          <Button
            key={item.key}
            variant="ghost"
            size="auto"
            type="button"
            role="tab"
            aria-selected={active}
            onClick={() => onChange(item.key)}
            className={cn(
              "inline-flex items-center gap-2 rounded-full border px-3.5 py-1.5",
              "text-[13px] font-semibold transition-colors duration-150",
              active
                ? "border-foreground bg-foreground text-background"
                : "border-border bg-card text-muted-foreground hover:border-border-strong hover:text-foreground",
            )}
          >
            {item.label}
            <span
              className={cn(
                "inline-flex min-w-[18px] items-center justify-center rounded-full px-1.5",
                "text-[10.5px] font-bold leading-none",
                active
                  ? "bg-background/20 text-background"
                  : "bg-border text-subtle-foreground",
              )}
            >
              {item.count}
            </span>
          </Button>
        )
      })}
    </div>
  )
}
