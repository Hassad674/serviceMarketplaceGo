"use client"

import { useTranslations } from "next-intl"
import { CountrySelect } from "@/shared/components/forms/country-select"
import {
  CityAutocomplete,
  type CitySelection,
} from "@/shared/components/forms/city-autocomplete"
import { NumberInput, SectionShell } from "./filter-primitives"

/**
 * FilterSectionLocation owns the geographic filters: country (single
 * select with flag emoji), city (Photon-backed combobox), and an
 * optional travel radius. Work mode lives in its own sibling section
 * (filter-section-work-mode.tsx) so the sidebar can hide it
 * independently for agency/referrer personas.
 *
 * The city combobox returns a `CitySelection` with lat/lon + canonical
 * country code, but the search filter only needs `city` + `countryCode`
 * as plain strings for Typesense — we pass them through directly and
 * keep lat/lon informational. Picking a city in a different country
 * also updates the country code so the two stay in sync.
 */
interface FilterSectionLocationProps {
  city: string
  countryCode: string
  radiusKm: number | null
  onCityChange: (next: string) => void
  onCountryChange: (next: string) => void
  onRadiusChange: (next: number | null) => void
}

export function FilterSectionLocation({
  city,
  countryCode,
  radiusKm,
  onCityChange,
  onCountryChange,
  onRadiusChange,
}: FilterSectionLocationProps) {
  const t = useTranslations("search.filters")

  // The combobox commits a CitySelection on Enter/click and passes
  // null when the user clears the input or switches country. We map
  // both events back to the parent-owned (city, countryCode) pair —
  // the search filter's contract only needs the strings.
  const selection: CitySelection | null =
    city.trim() === "" || countryCode.trim() === ""
      ? null
      : {
          city,
          countryCode,
          // Lat/lon are not part of the persisted filter shape; the
          // backend only matches on `city` + `country_code` strings.
          // We thread harmless 0/0 here so the shared CitySelection
          // type stays honoured downstream.
          latitude: 0,
          longitude: 0,
        }

  const handleCityChange = (next: CitySelection | null) => {
    if (next === null) {
      onCityChange("")
      return
    }
    if (next.countryCode && next.countryCode !== countryCode) {
      onCountryChange(next.countryCode)
    }
    onCityChange(next.city)
  }

  return (
    <SectionShell title={t("location")}>
      <CountrySelect
        value={countryCode}
        onChange={onCountryChange}
        placeholder={t("countryPlaceholder")}
        ariaLabel={t("countryPlaceholder")}
      />
      <CityAutocomplete
        value={selection}
        countryCode={countryCode}
        onChange={handleCityChange}
      />
      <NumberInput
        placeholder={t("radiusPlaceholder")}
        value={radiusKm}
        onChange={onRadiusChange}
        ariaLabel={t("radiusPlaceholder")}
      />
    </SectionShell>
  )
}
