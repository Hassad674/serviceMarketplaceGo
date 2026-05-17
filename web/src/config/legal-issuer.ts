// Single source of truth for the platform's legal identity (LCEN art.
// 6-III "Éditeur" block, RGPD data-controller block, invoicing issuer).
//
// The legal identity NEVER translates — a SIRET, a VAT number, a postal
// address are the same in every language. So this lives in plain TS,
// NOT in messages/{fr,en}.json (which would force a fragile duplicate
// that drifts between locales).
//
// Env-driven, fail-safe: every field reads a NEXT_PUBLIC_LEGAL_* env
// var and falls back to the real production value. Setting the env on
// Vercel lets us correct the identity (e.g. address change) WITHOUT a
// code deploy; the defaults guarantee the public pages are never blank
// even if the env is unset. Mirrors the backend INVOICE_ISSUER_* env
// contract so a single change on both platforms keeps invoices and
// public legal pages in sync.
//
// IMPORTANT — entity is a French micro-entreprise (entrepreneur
// individuel) under the "franchise en base de TVA" regime (art. 293 B
// CGI): invoices are issued HT with NO VAT charged. The intra-EU VAT
// number exists (required for some EU operations) but VAT is not
// applied while the franchise regime is in force.

function env(key: string, fallback: string): string {
  const value = process.env[key]
  return value && value.trim() !== "" ? value.trim() : fallback
}

export const legalIssuer = {
  // Raison sociale légale (entrepreneur individuel) + nom commercial.
  legalName: env("NEXT_PUBLIC_LEGAL_NAME", "Hassad SMARA — Entrepreneur Individuel (EI)"),
  tradingName: env("NEXT_PUBLIC_LEGAL_TRADING_NAME", "DesignedTrust Services"),
  legalForm: env(
    "NEXT_PUBLIC_LEGAL_FORM",
    "Micro-entreprise (entrepreneur individuel)",
  ),
  siret: env("NEXT_PUBLIC_LEGAL_SIRET", "87891296300021"),
  siren: env("NEXT_PUBLIC_LEGAL_SIREN", "878912963"),
  vatNumber: env("NEXT_PUBLIC_LEGAL_VAT", "FR26878912963"),
  apeCode: env("NEXT_PUBLIC_LEGAL_APE", "6201Z"),
  apeLabel: env(
    "NEXT_PUBLIC_LEGAL_APE_LABEL",
    "Programmation informatique",
  ),
  addressLine1: env("NEXT_PUBLIC_LEGAL_ADDRESS_LINE1", "254 rue Vendôme"),
  postalCode: env("NEXT_PUBLIC_LEGAL_POSTAL_CODE", "69003"),
  city: env("NEXT_PUBLIC_LEGAL_CITY", "Lyon 3"),
  country: env("NEXT_PUBLIC_LEGAL_COUNTRY", "France"),
  // RCS exemption text — micro-entreprise is dispensée d'immatriculation.
  rcsMention: env(
    "NEXT_PUBLIC_LEGAL_RCS_MENTION",
    "Dispensé d'immatriculation au RCS et au RM (micro-entreprise — art. L.123-1-1 du Code de commerce)",
  ),
  contactEmail: env(
    "NEXT_PUBLIC_LEGAL_CONTACT_EMAIL",
    "hassadsmara@designedtrust.com",
  ),
  dpoEmail: env("NEXT_PUBLIC_LEGAL_DPO_EMAIL", "dpo@designedtrust.com"),
  domain: env("NEXT_PUBLIC_LEGAL_DOMAIN", "services.designedtrust.com"),
} as const

// Pre-composed display strings used by the mentions-légales page so the
// page stays presentational. Identity is never translated; only the
// surrounding UI chrome (labels) goes through next-intl.
export const legalIssuerDisplay = {
  // "254 rue Vendôme, 69003 Lyon 3, France"
  fullAddress: `${legalIssuer.addressLine1}, ${legalIssuer.postalCode} ${legalIssuer.city}, ${legalIssuer.country}`,
  // "SIREN 878912963 — SIRET 87891296300021"
  registration: `SIREN ${legalIssuer.siren} — SIRET ${legalIssuer.siret}`,
  // "FR26878912963 (TVA non applicable, art. 293 B du CGI — franchise en base)"
  vatLine: `${legalIssuer.vatNumber} — TVA non applicable, art. 293 B du CGI (franchise en base : aucune TVA n'est facturée tant que ce régime s'applique).`,
  // "6201Z — Programmation informatique"
  apeLine: `${legalIssuer.apeCode} — ${legalIssuer.apeLabel}`,
} as const
