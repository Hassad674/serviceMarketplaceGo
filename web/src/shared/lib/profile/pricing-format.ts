// Supported fiat currencies for the profile pricing blocks. "pct" is
// the synthetic currency used exclusively by commission_pct rows — its
// amounts are stored in basis points (1/100 of a percent).
export const SUPPORTED_FIAT_CURRENCIES = [
  "EUR",
  "USD",
  "GBP",
  "CAD",
  "AUD",
] as const

export type FiatCurrency = (typeof SUPPORTED_FIAT_CURRENCIES)[number]

export type PricingLocale = "fr" | "en"

// FormattablePricingType is the widest union the formatter understands.
// The freelance and referrer pricing rows use disjoint subsets of this
// union, so the formatter can serve both persona features from a single
// implementation without either feature importing the other's types.
export type FormattablePricingType =
  | "daily"
  | "hourly"
  | "project_from"
  | "project_range"
  | "commission_pct"
  | "commission_flat"

// FormattablePricing is the minimal pricing shape the formatter needs.
// Features pass their own typed rows here — both FreelancePricing and
// ReferrerPricing satisfy this interface by construction.
export interface FormattablePricing {
  type: FormattablePricingType
  min_amount: number
  max_amount: number | null
  currency: string
}

// Smallest unit divisor. All fiat currencies we support are subdivided
// into hundredths (cents, pence, centimes...). commission_pct is
// special: its "smallest unit" is a basis point, so callers divide by
// 100 to go from stored value to percent.
const UNIT_DIVISOR_FIAT = 100
const UNIT_DIVISOR_PCT = 100

// Formats the *amount* portion only (no unit suffix). Amounts are
// converted from their stored smallest unit to human-readable display
// units before going through Intl.NumberFormat so we get proper locale
// grouping and decimal handling.
function formatAmount(
  amount: number,
  currency: string,
  locale: PricingLocale,
): string {
  const bcp = locale === "fr" ? "fr-FR" : "en-US"
  if (currency === "pct") {
    const pct = amount / UNIT_DIVISOR_PCT
    return new Intl.NumberFormat(bcp, {
      minimumFractionDigits: 0,
      maximumFractionDigits: 2,
    }).format(pct)
  }
  const value = amount / UNIT_DIVISOR_FIAT
  return new Intl.NumberFormat(bcp, {
    style: "currency",
    currency,
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  }).format(value)
}

function suffixForType(
  type: FormattablePricingType,
  locale: PricingLocale,
): string {
  if (locale === "fr") {
    switch (type) {
      case "daily":
        return "/j"
      case "hourly":
        return "/h"
      case "commission_flat":
        return " / deal"
      default:
        return ""
    }
  }
  switch (type) {
    case "daily":
      return "/day"
    case "hourly":
      return "/hr"
    case "commission_flat":
      return " per deal"
    default:
      return ""
  }
}

function prefixForType(
  type: FormattablePricingType,
  locale: PricingLocale,
): string {
  if (type !== "project_from") return ""
  return locale === "fr" ? "À partir de " : "From "
}

// Formats a pricing row into its canonical user-facing string.
//
// Examples (fr locale):
//   daily            500 €/j
//   hourly           75 €/h
//   project_from     À partir de 10 000 €
//   project_range    15 000 – 50 000 €
//   commission_pct   5 – 15 %
//   commission_flat  3 000 € / deal
//
// The pct type gets a trailing " %" suffix instead of a currency code.
// When max_amount is null on a range type we degrade gracefully by
// showing only the minimum — the backend validator should reject this
// combination but we stay defensive.
export function formatPricing(
  row: FormattablePricing,
  locale: PricingLocale = "fr",
): string {
  const min = formatAmount(row.min_amount, row.currency, locale)
  const hasMax = row.max_amount !== null && row.max_amount !== undefined
  const max = hasMax
    ? formatAmount(row.max_amount as number, row.currency, locale)
    : ""
  const prefix = prefixForType(row.type, locale)
  const suffix = suffixForType(row.type, locale)

  if (row.type === "commission_pct") {
    const body = hasMax ? `${min} – ${max}` : min
    return `${body} %`
  }

  if (row.type === "project_range" && hasMax) {
    // Avoid printing the currency symbol twice. Strip it from the min
    // side, keep it on the max side. Works for both "500 €" (fr) and
    // "€500" (en) layouts by locating the symbol position heuristically.
    const stripped = stripCurrencySymbol(min, locale)
    return `${stripped} – ${max}${suffix}`
  }

  return `${prefix}${min}${suffix}`
}

// Best-effort removal of the currency symbol from the min-side of a
// range so "15 000 € – 50 000 €" renders as "15 000 – 50 000 €".
// Locale-aware: symbol is trailing in fr, leading in en.
function stripCurrencySymbol(
  minFormatted: string,
  locale: PricingLocale,
): string {
  if (locale === "fr") {
    return minFormatted.replace(/\s*[^\d\s,.-].*$/u, "").trimEnd()
  }
  return minFormatted.replace(/^[^\d-]+/u, "")
}
