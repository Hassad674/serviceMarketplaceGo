"use client"

import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import type { SearchDocumentPersona } from "@/shared/lib/search/search-document"
import {
  EMPTY_SEARCH_FILTERS,
  isEmptyFilters,
  type SearchFilters,
} from "./search-filters"
import { FilterSectionAvailability } from "./filter-section-availability"
import { FilterSectionPricing } from "./filter-section-pricing"
import { FilterSectionLocation } from "./filter-section-location"
import { FilterSectionSkillsExpertise } from "./filter-section-skills-expertise"
import { FilterSectionRating } from "./filter-section-rating"

// SearchFilterSidebar renders the Malt-style left rail filter UI. It
// is intentionally logic-free: every change flows through `onChange`
// and the parent owns the state. The "Apply" button is a no-op today
// (it calls `onApply` which the parent may wire into Typesense later)
// and the "Reset" button re-emits the canonical empty state.
//
// Every section is labelled, keyboard-accessible, and uses the design
// system's semantic tokens — zero hardcoded colors. The actual
// rendering of each section is delegated to filter-section-*.tsx
// neighbors so this file stays focused on composition + layout.

interface SearchFilterSidebarProps {
  filters: SearchFilters
  onChange: (next: SearchFilters) => void
  onApply?: () => void
  resultsCount?: number
  className?: string
  /**
   * persona drives the price section's labels and unit suffix:
   *   - freelance -> "TJM min / max",  suffix "€"
   *   - agency    -> "Budget min / max", suffix "€"
   *   - referrer  -> "Commission min / max", suffix "%"
   *
   * Undefined falls back to the generic "Price min / max" labels —
   * this keeps the sidebar usable in contexts that never picked a
   * persona (e.g. stories / legacy callers). The min / max input
   * values are still raw numbers; the backend filter builder maps
   * them to the correct Typesense clause per persona.
   */
  persona?: SearchDocumentPersona
}

export function SearchFilterSidebar({
  filters,
  onChange,
  onApply,
  resultsCount,
  className,
  persona,
}: SearchFilterSidebarProps) {
  const t = useTranslations("search.filters")
  const tSearch = useTranslations("search")
  const hasFilters = !isEmptyFilters(filters)

  const update = <K extends keyof SearchFilters>(key: K, value: SearchFilters[K]) => {
    onChange({ ...filters, [key]: value })
  }

  return (
    <aside
      className={cn(
        "flex w-full flex-col gap-6 rounded-2xl border border-border bg-card p-5 shadow-sm",
        "lg:sticky lg:top-24 lg:max-h-[calc(100vh-7rem)] lg:overflow-y-auto",
        className,
      )}
      aria-label={t("title")}
    >
      <header className="flex items-center justify-between gap-2">
        <h2 className="text-base font-semibold text-foreground">{t("title")}</h2>
        {typeof resultsCount === "number" ? (
          <span className="text-xs text-muted-foreground">
            {tSearch("resultsCount", { count: resultsCount })}
          </span>
        ) : null}
      </header>

      <FilterSectionAvailability
        value={filters.availability}
        onChange={(v) => update("availability", v)}
      />
      <FilterSectionPricing
        persona={persona}
        min={filters.priceMin}
        max={filters.priceMax}
        onMinChange={(v) => update("priceMin", v)}
        onMaxChange={(v) => update("priceMax", v)}
      />
      <FilterSectionLocation
        city={filters.city}
        countryCode={filters.countryCode}
        radiusKm={filters.radiusKm}
        workModes={filters.workModes}
        onCityChange={(v) => update("city", v)}
        onCountryChange={(v) => update("countryCode", v)}
        onRadiusChange={(v) => update("radiusKm", v)}
        onWorkModesChange={(v) => update("workModes", v)}
      />
      <FilterSectionSkillsExpertise
        languages={filters.languages}
        expertise={filters.expertise}
        skills={filters.skills}
        onLanguagesChange={(v) => update("languages", v)}
        onExpertiseChange={(v) =>
          update("expertise", v as SearchFilters["expertise"])
        }
        onSkillsChange={(v) => update("skills", v)}
      />
      <FilterSectionRating
        value={filters.minRating}
        onChange={(v) => update("minRating", v)}
      />

      <footer className="sticky bottom-0 flex flex-col gap-2 bg-card pt-2">
        <button
          type="button"
          onClick={onApply}
          className="inline-flex h-10 items-center justify-center rounded-lg bg-rose-500 px-4 text-sm font-medium text-white transition-all duration-200 ease-out hover:bg-rose-600 hover:shadow-glow active:scale-[0.98] focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-rose-500/20"
        >
          {t("apply")}
        </button>
        {hasFilters ? (
          <button
            type="button"
            onClick={() => onChange(EMPTY_SEARCH_FILTERS)}
            className="inline-flex h-10 items-center justify-center rounded-lg border border-border bg-background px-4 text-sm font-medium text-muted-foreground transition-colors hover:text-foreground focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-rose-500/20"
          >
            {t("reset")}
          </button>
        ) : null}
      </footer>
    </aside>
  )
}
