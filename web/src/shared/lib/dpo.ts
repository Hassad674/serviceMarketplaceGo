// dpo.ts — single source of truth for the GDPR / RGPD point-of-contact
// email used across legal pages, the footer, and (eventually) the
// privacy policy MDX.
//
// Source: NEXT_PUBLIC_DPO_EMAIL env var. Falls back to the founder
// inbox so dev environments always have a usable mailto: link without
// requiring extra env wiring. Production deploys MUST set the env var
// to the official designated address.

const DEFAULT_DPO_EMAIL = "hassad.smara69@gmail.com"

/**
 * Resolve the configured DPO email. Reads NEXT_PUBLIC_DPO_EMAIL when
 * present, otherwise falls back to the founder address. Trim and
 * lower-case so consumers can rely on a stable canonical form for
 * mailto: links and on-page text.
 */
export function getDpoEmail(): string {
  const raw = process.env.NEXT_PUBLIC_DPO_EMAIL
  const trimmed = typeof raw === "string" ? raw.trim().toLowerCase() : ""
  if (trimmed.length > 0 && trimmed.includes("@")) {
    return trimmed
  }
  return DEFAULT_DPO_EMAIL
}
