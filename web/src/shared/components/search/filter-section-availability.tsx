"use client"

import { useTranslations } from "next-intl"
import type { SearchAvailabilityFilter } from "./search-filters"
import { PillButton, SectionShell } from "./filter-primitives"

const AVAILABILITY_OPTIONS: readonly SearchAvailabilityFilter[] = [
  "now",
  "soon",
  "all",
]

interface FilterSectionAvailabilityProps {
  value: SearchAvailabilityFilter
  onChange: (next: SearchAvailabilityFilter) => void
}

export function FilterSectionAvailability({
  value,
  onChange,
}: FilterSectionAvailabilityProps) {
  const t = useTranslations("search.filters")
  const labels: Record<SearchAvailabilityFilter, string> = {
    now: t("availableNow"),
    soon: t("availableSoon"),
    all: t("availabilityAll"),
  }
  return (
    <SectionShell title={t("availability")}>
      <div className="flex flex-wrap gap-2">
        {AVAILABILITY_OPTIONS.map((option) => (
          <PillButton
            key={option}
            label={labels[option]}
            selected={value === option}
            onClick={() => onChange(option)}
          />
        ))}
      </div>
    </SectionShell>
  )
}
