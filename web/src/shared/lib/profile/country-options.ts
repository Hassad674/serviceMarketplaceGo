// Curated list of countries commonly represented on the marketplace.
// Keys are ISO 3166-1 alpha-2 codes stored on the backend. Labels are
// localized here instead of through next-intl because this catalog is
// small and stable — we want zero runtime i18n lookups per render.

export type CountryOption = {
  code: string
  labelFr: string
  labelEn: string
}

export const COUNTRY_OPTIONS: readonly CountryOption[] = [
  { code: "FR", labelFr: "France", labelEn: "France" },
  { code: "BE", labelFr: "Belgique", labelEn: "Belgium" },
  { code: "CH", labelFr: "Suisse", labelEn: "Switzerland" },
  { code: "LU", labelFr: "Luxembourg", labelEn: "Luxembourg" },
  { code: "DE", labelFr: "Allemagne", labelEn: "Germany" },
  { code: "ES", labelFr: "Espagne", labelEn: "Spain" },
  { code: "IT", labelFr: "Italie", labelEn: "Italy" },
  { code: "PT", labelFr: "Portugal", labelEn: "Portugal" },
  { code: "NL", labelFr: "Pays-Bas", labelEn: "Netherlands" },
  { code: "IE", labelFr: "Irlande", labelEn: "Ireland" },
  { code: "GB", labelFr: "Royaume-Uni", labelEn: "United Kingdom" },
  { code: "DK", labelFr: "Danemark", labelEn: "Denmark" },
  { code: "SE", labelFr: "Suède", labelEn: "Sweden" },
  { code: "NO", labelFr: "Norvège", labelEn: "Norway" },
  { code: "FI", labelFr: "Finlande", labelEn: "Finland" },
  { code: "PL", labelFr: "Pologne", labelEn: "Poland" },
  { code: "US", labelFr: "États-Unis", labelEn: "United States" },
  { code: "CA", labelFr: "Canada", labelEn: "Canada" },
  { code: "AU", labelFr: "Australie", labelEn: "Australia" },
  { code: "BR", labelFr: "Brésil", labelEn: "Brazil" },
  { code: "MA", labelFr: "Maroc", labelEn: "Morocco" },
  { code: "TN", labelFr: "Tunisie", labelEn: "Tunisia" },
  { code: "SN", labelFr: "Sénégal", labelEn: "Senegal" },
] as const

const COUNTRY_CODE_SET = new Set(COUNTRY_OPTIONS.map((c) => c.code))

export function isKnownCountryCode(code: string): boolean {
  return COUNTRY_CODE_SET.has(code)
}

export function getCountryLabel(
  code: string,
  locale: "fr" | "en",
): string {
  const hit = COUNTRY_OPTIONS.find((c) => c.code === code)
  if (!hit) return code
  return locale === "fr" ? hit.labelFr : hit.labelEn
}

// Regional Indicator Symbols A-Z sit at code points 0x1F1E6..0x1F1FF.
// An alpha-2 country code maps to its flag by offsetting each letter
// from 'A' (0x41) onto that base. Pure string operation, no Intl, no
// emoji library — works on any modern runtime that supports the full
// Unicode plane.
export function getFlagEmoji(code: string): string {
  if (!code || code.length !== 2) return ""
  const upper = code.toUpperCase()
  const first = upper.charCodeAt(0)
  const second = upper.charCodeAt(1)
  if (first < 65 || first > 90 || second < 65 || second > 90) return ""
  return (
    String.fromCodePoint(0x1f1e6 + (first - 65)) +
    String.fromCodePoint(0x1f1e6 + (second - 65))
  )
}
