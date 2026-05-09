"use client"

import { useTranslations } from "next-intl"
import type { SearchWorkMode } from "./search-filters"
import { PillButton, SectionShell, toggle } from "./filter-primitives"

const WORK_MODE_OPTIONS: readonly SearchWorkMode[] = [
  "remote",
  "on_site",
  "hybrid",
]

/**
 * FilterSectionWorkMode renders the remote / on-site / hybrid toggle
 * that used to live next to the geographic filters. Splitting it out
 * lets the parent hide the whole section per persona — agencies and
 * referrers do not expose a per-engagement work-mode flag, only
 * freelances do.
 *
 * The visibility decision is owned by the parent (search-filter-sidebar).
 * This file is intentionally small so the conditional render at the
 * call site is `{visible.workMode ? <FilterSectionWorkMode .../> : null}`.
 */
interface FilterSectionWorkModeProps {
  workModes: SearchWorkMode[]
  onWorkModesChange: (next: SearchWorkMode[]) => void
}

export function FilterSectionWorkMode({
  workModes,
  onWorkModesChange,
}: FilterSectionWorkModeProps) {
  const t = useTranslations("search.filters")
  const labels: Record<SearchWorkMode, string> = {
    remote: t("remote"),
    on_site: t("onSite"),
    hybrid: t("hybrid"),
  }
  return (
    <SectionShell title={t("workMode")}>
      <div className="flex flex-wrap gap-2">
        {WORK_MODE_OPTIONS.map((mode) => (
          <PillButton
            key={mode}
            label={labels[mode]}
            selected={workModes.includes(mode)}
            onClick={() => onWorkModesChange(toggle(workModes, mode))}
          />
        ))}
      </div>
    </SectionShell>
  )
}
