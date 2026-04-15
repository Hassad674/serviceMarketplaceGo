// formatTotalEarned renders a human-friendly "total earned" line for
// the search card, Upwork-style. The amount is expected in the
// smallest unit (centimes / cents) — the same scale that
// SearchDocument.total_earned uses.
//
// The function refuses to render anything when the amount is zero or
// negative: the card hides the whole line in that case rather than
// showing "0 € gagnés" which would be visually noisy for every new
// profile.

import type { SearchDocumentPricing } from "./search-document"

export type FormatLocale = "fr" | "en"

const UNIT_DIVISOR = 100

function bcpLocale(locale: FormatLocale): string {
  return locale === "fr" ? "fr-FR" : "en-US"
}

// formatTotalEarned returns the localized amount with currency symbol,
// or an empty string when the amount is zero or negative.
export function formatTotalEarned(
  amountInSmallestUnit: number,
  currency: string,
  locale: FormatLocale,
): string {
  if (!Number.isFinite(amountInSmallestUnit) || amountInSmallestUnit <= 0) {
    return ""
  }
  const value = amountInSmallestUnit / UNIT_DIVISOR
  return new Intl.NumberFormat(bcpLocale(locale), {
    style: "currency",
    currency: currency || "EUR",
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  }).format(value)
}

// currencyForPricing extracts a fiat currency code from a pricing row,
// falling back to EUR when the row is missing or uses the dimensionless
// "pct" currency (which only applies to commission_pct).
export function currencyForPricing(
  pricing: SearchDocumentPricing | null | undefined,
): string {
  if (!pricing) return "EUR"
  if (pricing.currency === "pct") return "EUR"
  return pricing.currency || "EUR"
}
