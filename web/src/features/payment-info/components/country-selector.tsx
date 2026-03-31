"use client"

import { Globe } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"

// All 45 Stripe-supported countries
const STRIPE_COUNTRIES = [
  { code: "AT", name: "Austria" },
  { code: "AU", name: "Australia" },
  { code: "BE", name: "Belgium" },
  { code: "BG", name: "Bulgaria" },
  { code: "BR", name: "Brazil" },
  { code: "CA", name: "Canada" },
  { code: "CH", name: "Switzerland" },
  { code: "CY", name: "Cyprus" },
  { code: "CZ", name: "Czech Republic" },
  { code: "DE", name: "Germany" },
  { code: "DK", name: "Denmark" },
  { code: "EE", name: "Estonia" },
  { code: "ES", name: "Spain" },
  { code: "FI", name: "Finland" },
  { code: "FR", name: "France" },
  { code: "GB", name: "United Kingdom" },
  { code: "GR", name: "Greece" },
  { code: "HK", name: "Hong Kong" },
  { code: "HR", name: "Croatia" },
  { code: "HU", name: "Hungary" },
  { code: "IE", name: "Ireland" },
  { code: "IN", name: "India" },
  { code: "IT", name: "Italy" },
  { code: "JP", name: "Japan" },
  { code: "LT", name: "Lithuania" },
  { code: "LU", name: "Luxembourg" },
  { code: "LV", name: "Latvia" },
  { code: "MT", name: "Malta" },
  { code: "MX", name: "Mexico" },
  { code: "MY", name: "Malaysia" },
  { code: "NL", name: "Netherlands" },
  { code: "NO", name: "Norway" },
  { code: "NZ", name: "New Zealand" },
  { code: "PL", name: "Poland" },
  { code: "PT", name: "Portugal" },
  { code: "RO", name: "Romania" },
  { code: "SE", name: "Sweden" },
  { code: "SG", name: "Singapore" },
  { code: "SI", name: "Slovenia" },
  { code: "SK", name: "Slovakia" },
  { code: "TH", name: "Thailand" },
  { code: "AE", name: "United Arab Emirates" },
  { code: "US", name: "United States" },
] as const

interface CountrySelectorProps {
  value: string
  onChange: (country: string) => void
}

export function CountrySelector({ value, onChange }: CountrySelectorProps) {
  const t = useTranslations("paymentInfo")

  return (
    <div className="rounded-2xl border border-slate-100 bg-white shadow-sm dark:border-slate-700 dark:bg-slate-800/80 overflow-hidden">
      <div className="h-1 bg-gradient-to-r from-blue-500 to-indigo-500" />
      <div className="p-6 space-y-4">
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-blue-100 dark:bg-blue-500/20">
            <Globe className="h-5 w-5 text-blue-600 dark:text-blue-400" strokeWidth={1.5} />
          </div>
          <div>
            <h2 className="text-base font-semibold text-slate-900 dark:text-white">
              {t("activityCountry")}
            </h2>
            <p className="text-xs text-slate-500 dark:text-slate-400">
              {t("activityCountryDesc")}
            </p>
          </div>
        </div>

        <select
          value={value}
          onChange={(e) => onChange(e.target.value)}
          className={cn(
            "h-10 w-full rounded-lg border border-slate-200 bg-white px-3 text-sm",
            "focus:border-rose-500 focus:ring-4 focus:ring-rose-500/10 focus:outline-none",
            "dark:border-slate-600 dark:bg-slate-800 dark:text-white",
          )}
        >
          <option value="">{t("selectCountry")}</option>
          {STRIPE_COUNTRIES.map((c) => (
            <option key={c.code} value={c.code}>
              {c.name}
            </option>
          ))}
        </select>
      </div>
    </div>
  )
}

/** Detect country from browser locale. */
export function detectBrowserCountry(): string {
  if (typeof navigator === "undefined") return ""
  const lang = navigator.language || ""
  // "fr-FR" → "FR", "en-US" → "US", "de" → "DE"
  const parts = lang.split("-")
  const candidate = (parts[1] ?? parts[0]).toUpperCase()
  if (candidate.length === 2 && STRIPE_COUNTRIES.some((c) => c.code === candidate)) {
    return candidate
  }
  return ""
}
