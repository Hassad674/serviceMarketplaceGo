// Thin shim re-exporting the shared language catalog so the legacy
// provider feature keeps working after the split-profile refactor.
// The canonical implementation lives in shared/lib/profile so the
// new freelance-profile and referrer-profile features can use it
// without crossing feature boundaries.
export {
  LANGUAGE_OPTIONS,
  isKnownLanguageCode,
  getLanguageLabel,
  getLanguageFlagCountry,
  type LanguageOption,
} from "@/shared/lib/profile/language-options"
