"use client"

import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"

const COUNTRIES = [
  { code: "FR", name: "France" },
  { code: "DE", name: "Germany" },
  { code: "ES", name: "Spain" },
  { code: "IT", name: "Italy" },
  { code: "PT", name: "Portugal" },
  { code: "NL", name: "Netherlands" },
  { code: "BE", name: "Belgium" },
  { code: "LU", name: "Luxembourg" },
  { code: "CH", name: "Switzerland" },
  { code: "AT", name: "Austria" },
  { code: "IE", name: "Ireland" },
  { code: "GB", name: "United Kingdom" },
  { code: "SE", name: "Sweden" },
  { code: "DK", name: "Denmark" },
  { code: "NO", name: "Norway" },
  { code: "FI", name: "Finland" },
  { code: "PL", name: "Poland" },
  { code: "CZ", name: "Czech Republic" },
  { code: "RO", name: "Romania" },
  { code: "GR", name: "Greece" },
  { code: "HR", name: "Croatia" },
  { code: "BG", name: "Bulgaria" },
  { code: "HU", name: "Hungary" },
  { code: "SK", name: "Slovakia" },
  { code: "SI", name: "Slovenia" },
  { code: "EE", name: "Estonia" },
  { code: "LV", name: "Latvia" },
  { code: "LT", name: "Lithuania" },
  { code: "CY", name: "Cyprus" },
  { code: "MT", name: "Malta" },
  { code: "US", name: "United States" },
  { code: "CA", name: "Canada" },
  { code: "AU", name: "Australia" },
  { code: "JP", name: "Japan" },
  { code: "SG", name: "Singapore" },
  { code: "IN", name: "India" },
  { code: "BR", name: "Brazil" },
  { code: "MX", name: "Mexico" },
  { code: "MA", name: "Morocco" },
  { code: "TN", name: "Tunisia" },
  { code: "SN", name: "Senegal" },
  { code: "CI", name: "Ivory Coast" },
]

/** Countries where IBAN is the standard bank account format. */
const IBAN_COUNTRIES = new Set([
  "FR", "DE", "ES", "IT", "PT", "NL", "BE", "LU", "CH", "AT", "IE", "GB",
  "SE", "DK", "NO", "FI", "PL", "CZ", "RO", "GR", "HR", "BG", "HU", "SK",
  "SI", "EE", "LV", "LT", "CY", "MT",
])

export function isIbanCountry(code: string): boolean {
  return IBAN_COUNTRIES.has(code)
}

type CountrySelectProps = {
  value: string
  onChange: (value: string) => void
  hasError?: boolean
}

export function CountrySelect({ value, onChange, hasError }: CountrySelectProps) {
  const t = useTranslations("paymentInfo")

  return (
    <select
      value={value}
      onChange={(e) => onChange(e.target.value)}
      aria-label={t("country")}
      className={cn(
        "h-10 w-full rounded-lg border bg-white px-3 text-sm shadow-xs transition-all duration-200",
        "focus:border-rose-500 focus:ring-4 focus:ring-rose-500/10 focus:outline-none",
        "dark:bg-gray-900 dark:text-gray-100",
        hasError
          ? "border-red-500 ring-4 ring-red-500/10"
          : "border-gray-200 dark:border-gray-700",
      )}
    >
      <option value="">{t("country")}</option>
      {COUNTRIES.map((c) => (
        <option key={c.code} value={c.code}>
          {c.name}
        </option>
      ))}
    </select>
  )
}
