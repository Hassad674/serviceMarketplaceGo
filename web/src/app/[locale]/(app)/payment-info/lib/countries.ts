/**
 * Countries Stripe Connect Custom accounts can be created in, FROM a FR
 * platform. Empirically verified via the Stripe API on 2026-04-05 — we
 * attempted to create a test account in every one of Stripe's 45 supported
 * countries and kept only the 43 that succeeded.
 *
 * Excluded (can't be created from a FR platform):
 *   - BR (cross-border restriction)
 *   - IN (cross-border restriction)
 * Company-only from FR (individual business type not supported):
 *   - AE (UAE) — flagged via `companyOnly: true`
 *
 * Each entry: ISO 3166-1 alpha-2 code, emoji flag, French label, English label, region.
 * Sorted by region then label for intuitive grouping in the selector.
 */

export type SupportedCountry = {
  code: string
  flag: string
  labelFr: string
  labelEn: string
  region: "eu" | "europe_other" | "americas" | "apac" | "mena"
  /**
   * When true, the country cannot onboard as an individual from a FR
   * platform — only as a company. Used to filter the business-type options.
   */
  companyOnly?: boolean
}

export const STRIPE_CONNECT_COUNTRIES: SupportedCountry[] = [
  // European Union
  { code: "FR", flag: "🇫🇷", labelFr: "France", labelEn: "France", region: "eu" },
  { code: "BE", flag: "🇧🇪", labelFr: "Belgique", labelEn: "Belgium", region: "eu" },
  { code: "LU", flag: "🇱🇺", labelFr: "Luxembourg", labelEn: "Luxembourg", region: "eu" },
  { code: "DE", flag: "🇩🇪", labelFr: "Allemagne", labelEn: "Germany", region: "eu" },
  { code: "AT", flag: "🇦🇹", labelFr: "Autriche", labelEn: "Austria", region: "eu" },
  { code: "NL", flag: "🇳🇱", labelFr: "Pays-Bas", labelEn: "Netherlands", region: "eu" },
  { code: "IT", flag: "🇮🇹", labelFr: "Italie", labelEn: "Italy", region: "eu" },
  { code: "ES", flag: "🇪🇸", labelFr: "Espagne", labelEn: "Spain", region: "eu" },
  { code: "PT", flag: "🇵🇹", labelFr: "Portugal", labelEn: "Portugal", region: "eu" },
  { code: "IE", flag: "🇮🇪", labelFr: "Irlande", labelEn: "Ireland", region: "eu" },
  { code: "DK", flag: "🇩🇰", labelFr: "Danemark", labelEn: "Denmark", region: "eu" },
  { code: "SE", flag: "🇸🇪", labelFr: "Suède", labelEn: "Sweden", region: "eu" },
  { code: "FI", flag: "🇫🇮", labelFr: "Finlande", labelEn: "Finland", region: "eu" },
  { code: "PL", flag: "🇵🇱", labelFr: "Pologne", labelEn: "Poland", region: "eu" },
  { code: "CZ", flag: "🇨🇿", labelFr: "République Tchèque", labelEn: "Czech Republic", region: "eu" },
  { code: "GR", flag: "🇬🇷", labelFr: "Grèce", labelEn: "Greece", region: "eu" },
  { code: "EE", flag: "🇪🇪", labelFr: "Estonie", labelEn: "Estonia", region: "eu" },
  { code: "LV", flag: "🇱🇻", labelFr: "Lettonie", labelEn: "Latvia", region: "eu" },
  { code: "LT", flag: "🇱🇹", labelFr: "Lituanie", labelEn: "Lithuania", region: "eu" },
  { code: "SK", flag: "🇸🇰", labelFr: "Slovaquie", labelEn: "Slovakia", region: "eu" },
  { code: "SI", flag: "🇸🇮", labelFr: "Slovénie", labelEn: "Slovenia", region: "eu" },
  { code: "HU", flag: "🇭🇺", labelFr: "Hongrie", labelEn: "Hungary", region: "eu" },
  { code: "BG", flag: "🇧🇬", labelFr: "Bulgarie", labelEn: "Bulgaria", region: "eu" },
  { code: "HR", flag: "🇭🇷", labelFr: "Croatie", labelEn: "Croatia", region: "eu" },
  { code: "RO", flag: "🇷🇴", labelFr: "Roumanie", labelEn: "Romania", region: "eu" },
  { code: "CY", flag: "🇨🇾", labelFr: "Chypre", labelEn: "Cyprus", region: "eu" },
  { code: "MT", flag: "🇲🇹", labelFr: "Malte", labelEn: "Malta", region: "eu" },

  // Europe (non-EU)
  { code: "GB", flag: "🇬🇧", labelFr: "Royaume-Uni", labelEn: "United Kingdom", region: "europe_other" },
  { code: "CH", flag: "🇨🇭", labelFr: "Suisse", labelEn: "Switzerland", region: "europe_other" },
  { code: "NO", flag: "🇳🇴", labelFr: "Norvège", labelEn: "Norway", region: "europe_other" },
  { code: "LI", flag: "🇱🇮", labelFr: "Liechtenstein", labelEn: "Liechtenstein", region: "europe_other" },
  { code: "GI", flag: "🇬🇮", labelFr: "Gibraltar", labelEn: "Gibraltar", region: "europe_other" },

  // Americas
  { code: "US", flag: "🇺🇸", labelFr: "États-Unis", labelEn: "United States", region: "americas" },
  { code: "CA", flag: "🇨🇦", labelFr: "Canada", labelEn: "Canada", region: "americas" },
  { code: "MX", flag: "🇲🇽", labelFr: "Mexique", labelEn: "Mexico", region: "americas" },
  // BR (Brazil) removed: cross-border restriction from FR platform

  // APAC
  { code: "AU", flag: "🇦🇺", labelFr: "Australie", labelEn: "Australia", region: "apac" },
  { code: "NZ", flag: "🇳🇿", labelFr: "Nouvelle-Zélande", labelEn: "New Zealand", region: "apac" },
  { code: "SG", flag: "🇸🇬", labelFr: "Singapour", labelEn: "Singapore", region: "apac" },
  { code: "HK", flag: "🇭🇰", labelFr: "Hong Kong", labelEn: "Hong Kong", region: "apac" },
  { code: "JP", flag: "🇯🇵", labelFr: "Japon", labelEn: "Japan", region: "apac" },
  // IN (India) removed: cross-border restriction from FR platform
  { code: "TH", flag: "🇹🇭", labelFr: "Thaïlande", labelEn: "Thailand", region: "apac" },
  { code: "MY", flag: "🇲🇾", labelFr: "Malaisie", labelEn: "Malaysia", region: "apac" },

  // MENA
  { code: "AE", flag: "🇦🇪", labelFr: "Émirats Arabes Unis", labelEn: "United Arab Emirates", region: "mena", companyOnly: true },
]

export const REGION_LABELS: Record<SupportedCountry["region"], string> = {
  eu: "Union Européenne",
  europe_other: "Europe",
  americas: "Amériques",
  apac: "Asie-Pacifique",
  mena: "Moyen-Orient",
}

export function findCountry(code: string): SupportedCountry | undefined {
  return STRIPE_CONNECT_COUNTRIES.find((c) => c.code === code.toUpperCase())
}

export function searchCountries(query: string, locale: "fr" | "en" = "fr"): SupportedCountry[] {
  const q = query.trim().toLowerCase()
  if (!q) return STRIPE_CONNECT_COUNTRIES
  return STRIPE_CONNECT_COUNTRIES.filter((c) => {
    const label = locale === "fr" ? c.labelFr : c.labelEn
    return (
      label.toLowerCase().includes(q) ||
      c.code.toLowerCase().includes(q)
    )
  })
}
