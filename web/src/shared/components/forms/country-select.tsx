"use client"

import { useId, useMemo } from "react"
import { useLocale } from "next-intl"
import { cn } from "@/shared/lib/utils"
import {
  COUNTRY_OPTIONS,
  getCountryLabel,
  getFlagEmoji,
} from "@/shared/lib/profile/country-options"

import { Select } from "@/shared/components/ui/select"

/**
 * CountrySelect is the single shared country picker used by both the
 * search filter sidebar AND the editable profile location section.
 * It wraps the Soleil v2 native `<Select>` primitive so the dropdown
 * inherits the corail focus ring + h-10 default and stays
 * keyboard-accessible for free.
 *
 * The list source is `COUNTRY_OPTIONS` (alpha-2 codes + FR/EN
 * labels) — small and stable, kept in `shared/lib/profile/` so both
 * apps (web + mobile via parity) can reuse the catalogue. The flag
 * emoji is derived at render time from the alpha-2 code via Unicode
 * Regional Indicator Symbols, so there is no asset payload added.
 *
 * Empty `""` is the canonical "nothing selected" value the parent
 * persists. The placeholder is forwarded to the native select via
 * the `placeholder` prop on the primitive.
 */
export interface CountrySelectProps {
  value: string
  onChange: (next: string) => void
  placeholder?: string
  label?: string
  disabled?: boolean
  className?: string
  id?: string
  /**
   * ariaLabel is forwarded as `aria-label` when the parent does not
   * pass a visible label — keeps screen-reader announcements correct
   * inside the search sidebar where the visual label lives one level
   * up.
   */
  ariaLabel?: string
}

export function CountrySelect({
  value,
  onChange,
  placeholder,
  label,
  disabled,
  className,
  id,
  ariaLabel,
}: CountrySelectProps) {
  const locale = useLocale() === "fr" ? "fr" : "en"
  const generatedId = useId()
  const selectId = id ?? generatedId

  // Pre-compute the option labels with the active locale so the
  // render path stays linear. The list is stable so memoising on the
  // locale alone is enough — no allocation per render.
  const options = useMemo(
    () =>
      COUNTRY_OPTIONS.map((option) => ({
        value: option.code,
        label: `${getFlagEmoji(option.code)} ${getCountryLabel(option.code, locale)}`,
      })),
    [locale],
  )

  return (
    <Select
      id={selectId}
      value={value}
      onChange={(event) => onChange(event.target.value)}
      placeholder={placeholder}
      label={label}
      aria-label={ariaLabel}
      disabled={disabled}
      options={options}
      className={cn("w-full", className)}
    />
  )
}
