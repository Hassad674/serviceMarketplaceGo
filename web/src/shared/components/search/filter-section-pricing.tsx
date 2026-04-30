"use client"

import { useTranslations } from "next-intl"
import type { SearchDocumentPersona } from "@/shared/lib/search/search-document"
import { NumberInput, SectionShell } from "./filter-primitives"

// FilterSectionPricing renders the min / max bounds the parent pipes
// into the Typesense filter_by builder. Labels and unit suffix are
// persona-aware (see buildPriceLabels) so the UX matches the primary
// pricing shape for the persona being searched. The input values
// stay raw numbers — the persona only affects how the bounds are
// labelled for the user, not how they are persisted or sent to the
// backend.

interface FilterSectionPricingProps {
  persona: SearchDocumentPersona | undefined
  min: number | null
  max: number | null
  onMinChange: (next: number | null) => void
  onMaxChange: (next: number | null) => void
}

export function FilterSectionPricing({
  persona,
  min,
  max,
  onMinChange,
  onMaxChange,
}: FilterSectionPricingProps) {
  const t = useTranslations("search.filters")
  const labels = buildPriceLabels(t, persona)
  return (
    <SectionShell title={labels.title}>
      <div className="flex items-center gap-2">
        <NumberInputWithSuffix
          placeholder={labels.minPlaceholder}
          ariaLabel={labels.minPlaceholder}
          suffix={labels.unit}
          value={min}
          onChange={onMinChange}
        />
        <span className="text-xs text-muted-foreground">–</span>
        <NumberInputWithSuffix
          placeholder={labels.maxPlaceholder}
          ariaLabel={labels.maxPlaceholder}
          suffix={labels.unit}
          value={max}
          onChange={onMaxChange}
        />
      </div>
    </SectionShell>
  )
}

interface PriceLabels {
  title: string
  minPlaceholder: string
  maxPlaceholder: string
  unit: string
}

// buildPriceLabels returns the persona-specific title / placeholders /
// unit suffix for the FilterSectionPricing. Undefined persona falls
// back to the generic price labels so legacy callers keep working.
export function buildPriceLabels(
  t: ReturnType<typeof useTranslations>,
  persona: SearchDocumentPersona | undefined,
): PriceLabels {
  switch (persona) {
    case "freelance":
      return {
        title: t("freelancePrice"),
        minPlaceholder: t("freelancePriceMin"),
        maxPlaceholder: t("freelancePriceMax"),
        unit: "€",
      }
    case "agency":
      return {
        title: t("agencyPrice"),
        minPlaceholder: t("agencyPriceMin"),
        maxPlaceholder: t("agencyPriceMax"),
        unit: "€",
      }
    case "referrer":
      return {
        title: t("referrerPrice"),
        minPlaceholder: t("referrerPriceMin"),
        maxPlaceholder: t("referrerPriceMax"),
        unit: "%",
      }
    default:
      return {
        title: t("price"),
        minPlaceholder: t("priceMin"),
        maxPlaceholder: t("priceMax"),
        unit: "",
      }
  }
}

// NumberInputWithSuffix is a NumberInput decorated with a trailing
// unit suffix (€ or %). When suffix is empty we fall back to the
// plain input so we do not reserve padding for nothing. Kept inside
// this file because the suffix is purely cosmetic to the price
// section — no other section needs it.
function NumberInputWithSuffix({
  value,
  onChange,
  placeholder,
  ariaLabel,
  suffix,
}: {
  value: number | null
  onChange: (next: number | null) => void
  placeholder: string
  ariaLabel: string
  suffix: string
}) {
  if (suffix === "") {
    return (
      <NumberInput
        value={value}
        onChange={onChange}
        placeholder={placeholder}
        ariaLabel={ariaLabel}
      />
    )
  }
  return (
    <div className="relative w-full min-w-0">
      <input
        type="number"
        min={0}
        inputMode="numeric"
        value={value ?? ""}
        placeholder={placeholder}
        aria-label={ariaLabel}
        onChange={(e) => {
          const raw = e.target.value.trim()
          onChange(raw === "" ? null : Math.max(0, Number(raw) || 0))
        }}
        className="h-10 w-full min-w-0 rounded-lg border border-border bg-background pl-3 pr-8 text-sm shadow-xs focus:border-rose-500 focus:outline-none focus:ring-4 focus:ring-rose-500/10"
      />
      <span
        aria-hidden="true"
        className="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2 text-xs font-medium text-muted-foreground"
      >
        {suffix}
      </span>
    </div>
  )
}
