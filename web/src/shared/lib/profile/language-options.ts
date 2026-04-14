// Curated ISO 639-1 language catalog. Same rationale as
// country-options.ts — small stable list, hand-localized so there is
// no runtime Intl.DisplayNames dependency (not universally available
// across all browsers we target).

export type LanguageOption = {
  code: string
  labelFr: string
  labelEn: string
  // Single flag emoji representing a "canonical" country for the
  // language. Used for compact display on cards/hero strip. English
  // falls back to the UK flag, Arabic to Morocco etc.
  flagCountryCode: string
}

export const LANGUAGE_OPTIONS: readonly LanguageOption[] = [
  { code: "fr", labelFr: "Français", labelEn: "French", flagCountryCode: "FR" },
  { code: "en", labelFr: "Anglais", labelEn: "English", flagCountryCode: "GB" },
  { code: "es", labelFr: "Espagnol", labelEn: "Spanish", flagCountryCode: "ES" },
  { code: "de", labelFr: "Allemand", labelEn: "German", flagCountryCode: "DE" },
  { code: "it", labelFr: "Italien", labelEn: "Italian", flagCountryCode: "IT" },
  { code: "pt", labelFr: "Portugais", labelEn: "Portuguese", flagCountryCode: "PT" },
  { code: "nl", labelFr: "Néerlandais", labelEn: "Dutch", flagCountryCode: "NL" },
  { code: "pl", labelFr: "Polonais", labelEn: "Polish", flagCountryCode: "PL" },
  { code: "ar", labelFr: "Arabe", labelEn: "Arabic", flagCountryCode: "MA" },
  { code: "zh", labelFr: "Chinois", labelEn: "Chinese", flagCountryCode: "CN" },
  { code: "ja", labelFr: "Japonais", labelEn: "Japanese", flagCountryCode: "JP" },
  { code: "ru", labelFr: "Russe", labelEn: "Russian", flagCountryCode: "RU" },
  { code: "tr", labelFr: "Turc", labelEn: "Turkish", flagCountryCode: "TR" },
  { code: "ko", labelFr: "Coréen", labelEn: "Korean", flagCountryCode: "KR" },
  { code: "hi", labelFr: "Hindi", labelEn: "Hindi", flagCountryCode: "IN" },
  { code: "sv", labelFr: "Suédois", labelEn: "Swedish", flagCountryCode: "SE" },
  { code: "no", labelFr: "Norvégien", labelEn: "Norwegian", flagCountryCode: "NO" },
  { code: "da", labelFr: "Danois", labelEn: "Danish", flagCountryCode: "DK" },
  { code: "fi", labelFr: "Finnois", labelEn: "Finnish", flagCountryCode: "FI" },
  { code: "cs", labelFr: "Tchèque", labelEn: "Czech", flagCountryCode: "CZ" },
  { code: "ro", labelFr: "Roumain", labelEn: "Romanian", flagCountryCode: "RO" },
  { code: "hu", labelFr: "Hongrois", labelEn: "Hungarian", flagCountryCode: "HU" },
  { code: "el", labelFr: "Grec", labelEn: "Greek", flagCountryCode: "GR" },
  { code: "he", labelFr: "Hébreu", labelEn: "Hebrew", flagCountryCode: "IL" },
  { code: "uk", labelFr: "Ukrainien", labelEn: "Ukrainian", flagCountryCode: "UA" },
  { code: "vi", labelFr: "Vietnamien", labelEn: "Vietnamese", flagCountryCode: "VN" },
  { code: "th", labelFr: "Thaï", labelEn: "Thai", flagCountryCode: "TH" },
  { code: "id", labelFr: "Indonésien", labelEn: "Indonesian", flagCountryCode: "ID" },
  { code: "ms", labelFr: "Malais", labelEn: "Malay", flagCountryCode: "MY" },
  { code: "sw", labelFr: "Swahili", labelEn: "Swahili", flagCountryCode: "KE" },
] as const

const LANGUAGE_CODE_SET = new Set(LANGUAGE_OPTIONS.map((l) => l.code))

export function isKnownLanguageCode(code: string): boolean {
  return LANGUAGE_CODE_SET.has(code)
}

export function getLanguageLabel(
  code: string,
  locale: "fr" | "en",
): string {
  const hit = LANGUAGE_OPTIONS.find((l) => l.code === code)
  if (!hit) return code
  return locale === "fr" ? hit.labelFr : hit.labelEn
}

export function getLanguageFlagCountry(code: string): string {
  const hit = LANGUAGE_OPTIONS.find((l) => l.code === code)
  return hit?.flagCountryCode ?? ""
}
