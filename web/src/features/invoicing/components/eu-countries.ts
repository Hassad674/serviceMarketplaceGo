// EU member states whose VAT numbers can be validated through VIES.
// 2-letter ISO codes — the country dropdown emits these directly so
// the form value is always wire-compatible with the backend.

export type EUCountryCode =
  | "FR"
  | "BE"
  | "DE"
  | "ES"
  | "IT"
  | "NL"
  | "PT"
  | "LU"
  | "IE"
  | "AT"
  | "PL"
  | "RO"
  | "CZ"
  | "SE"
  | "FI"
  | "DK"
  | "EL" // Greece — VIES uses EL, not GR
  | "HU"
  | "BG"
  | "HR"
  | "SI"
  | "SK"
  | "EE"
  | "LV"
  | "LT"
  | "MT"
  | "CY"

export const EU_COUNTRIES: { code: EUCountryCode; label: string }[] = [
  { code: "FR", label: "France" },
  { code: "BE", label: "Belgique" },
  { code: "DE", label: "Allemagne" },
  { code: "ES", label: "Espagne" },
  { code: "IT", label: "Italie" },
  { code: "NL", label: "Pays-Bas" },
  { code: "PT", label: "Portugal" },
  { code: "LU", label: "Luxembourg" },
  { code: "IE", label: "Irlande" },
  { code: "AT", label: "Autriche" },
  { code: "PL", label: "Pologne" },
  { code: "RO", label: "Roumanie" },
  { code: "CZ", label: "République tchèque" },
  { code: "SE", label: "Suède" },
  { code: "FI", label: "Finlande" },
  { code: "DK", label: "Danemark" },
  { code: "EL", label: "Grèce" },
  { code: "HU", label: "Hongrie" },
  { code: "BG", label: "Bulgarie" },
  { code: "HR", label: "Croatie" },
  { code: "SI", label: "Slovénie" },
  { code: "SK", label: "Slovaquie" },
  { code: "EE", label: "Estonie" },
  { code: "LV", label: "Lettonie" },
  { code: "LT", label: "Lituanie" },
  { code: "MT", label: "Malte" },
  { code: "CY", label: "Chypre" },
]

const EU_SET = new Set<string>(EU_COUNTRIES.map((c) => c.code))

export function isEUCountry(code: string): boolean {
  return EU_SET.has(code)
}
