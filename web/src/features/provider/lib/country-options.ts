// Thin shim re-exporting the shared country catalog so the legacy
// provider feature keeps working after the split-profile refactor.
// The canonical implementation lives in shared/lib/profile so the
// new freelance-profile and referrer-profile features can use it
// without crossing feature boundaries.
export {
  COUNTRY_OPTIONS,
  isKnownCountryCode,
  getCountryLabel,
  getFlagEmoji,
  type CountryOption,
} from "@/shared/lib/profile/country-options"
