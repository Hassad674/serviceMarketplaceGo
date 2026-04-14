// Thin shim re-exporting the shared pricing formatter so the legacy
// provider feature keeps working after the split-profile refactor.
// The canonical implementation lives in shared/lib/profile so the
// new freelance-profile and referrer-profile features can use it
// without crossing feature boundaries.
//
// TODO: when the agency profile is refactored onto the split-profile
// backend, move agency consumers to shared/lib/profile directly and
// delete this shim.

import type { Pricing } from "../api/profile-api"
import {
  formatPricing as sharedFormatPricing,
  type FormattablePricing,
  type PricingLocale,
} from "@/shared/lib/profile/pricing-format"

export {
  SUPPORTED_FIAT_CURRENCIES,
  type FiatCurrency,
  type PricingLocale,
} from "@/shared/lib/profile/pricing-format"

// formatPricing is the legacy signature used by the provider feature:
// it takes the feature's own Pricing shape (which carries `kind` on
// top of the shared fields). kind is not read by the formatter, so we
// simply forward to the shared implementation.
export function formatPricing(row: Pricing, locale: PricingLocale = "fr"): string {
  return sharedFormatPricing(row as FormattablePricing, locale)
}
