"use client"

import { useTranslations } from "next-intl"
import type { SearchWorkMode } from "./search-filters"
import {
  NumberInput,
  PillButton,
  SectionShell,
  toggle,
} from "./filter-primitives"

const WORK_MODE_OPTIONS: readonly SearchWorkMode[] = [
  "remote",
  "on_site",
  "hybrid",
]

// Combines the geographic filters (city + country + radius) with the
// remote/on-site/hybrid work-mode toggle so the "where do you work"
// concerns live next to each other. The parent still owns the state.

interface FilterSectionLocationProps {
  city: string
  countryCode: string
  radiusKm: number | null
  workModes: SearchWorkMode[]
  onCityChange: (next: string) => void
  onCountryChange: (next: string) => void
  onRadiusChange: (next: number | null) => void
  onWorkModesChange: (next: SearchWorkMode[]) => void
}

export function FilterSectionLocation({
  city,
  countryCode,
  radiusKm,
  workModes,
  onCityChange,
  onCountryChange,
  onRadiusChange,
  onWorkModesChange,
}: FilterSectionLocationProps) {
  const t = useTranslations("search.filters")
  const workModeLabels: Record<SearchWorkMode, string> = {
    remote: t("remote"),
    on_site: t("onSite"),
    hybrid: t("hybrid"),
  }
  return (
    <>
      <SectionShell title={t("location")}>
        <input
          type="text"
          value={city}
          onChange={(e) => onCityChange(e.target.value)}
          placeholder={t("cityPlaceholder")}
          aria-label={t("cityPlaceholder")}
          className="h-10 rounded-lg border border-border bg-background px-3 text-sm shadow-xs focus:border-rose-500 focus:outline-none focus:ring-4 focus:ring-rose-500/10"
        />
        <input
          type="text"
          value={countryCode}
          onChange={(e) =>
            onCountryChange(e.target.value.toUpperCase().slice(0, 2))
          }
          placeholder={t("countryPlaceholder")}
          aria-label={t("countryPlaceholder")}
          maxLength={2}
          className="h-10 w-20 rounded-lg border border-border bg-background px-3 text-sm uppercase shadow-xs focus:border-rose-500 focus:outline-none focus:ring-4 focus:ring-rose-500/10"
        />
        <NumberInput
          placeholder={t("radiusPlaceholder")}
          value={radiusKm}
          onChange={onRadiusChange}
          ariaLabel={t("radiusPlaceholder")}
        />
      </SectionShell>
      <SectionShell title={t("workMode")}>
        <div className="flex flex-wrap gap-2">
          {WORK_MODE_OPTIONS.map((mode) => (
            <PillButton
              key={mode}
              label={workModeLabels[mode]}
              selected={workModes.includes(mode)}
              onClick={() => onWorkModesChange(toggle(workModes, mode))}
            />
          ))}
        </div>
      </SectionShell>
    </>
  )
}
