// Frozen catalog of expertise domain keys supported by the marketplace.
// These strings are the source of truth shared with the backend — every
// domain key sent to PUT /api/v1/profile/expertise must belong to this
// list. The i18n layer maps each key to a localized label.
//
// Ordering here reflects the default picker order in the UI. The user's
// saved selection has its own persisted ordering and is always rendered
// in that order on the profile — this list is only consulted when the
// editor renders the pool of all available domains.
export const EXPERTISE_DOMAIN_KEYS = [
  "development",
  "data_ai_ml",
  "design_ui_ux",
  "design_3d_animation",
  "video_motion",
  "photo_audiovisual",
  "marketing_growth",
  "writing_translation",
  "business_dev_sales",
  "consulting_strategy",
  "product_ux_research",
  "ops_admin_support",
  "legal",
  "finance_accounting",
  "hr_recruitment",
] as const

export type ExpertiseDomainKey = (typeof EXPERTISE_DOMAIN_KEYS)[number]

// Narrows an unknown string into a known domain key — used to filter out
// unexpected values coming from the API (forward-compat with backends
// that may ship new keys before the frontend knows about them).
export function isExpertiseDomainKey(value: string): value is ExpertiseDomainKey {
  return (EXPERTISE_DOMAIN_KEYS as readonly string[]).includes(value)
}

// Per-org-type maximum number of expertise domains that may be selected.
// Mirrors backend validation — if these diverge the UI will still cap
// the picker, and the backend will reject oversize payloads with 400.
// `enterprise` has no expertise section at all, hence 0.
const EXPERTISE_MAX_BY_ORG_TYPE: Record<string, number> = {
  agency: 8,
  provider_personal: 5,
  enterprise: 0,
}

export function getMaxExpertiseForOrgType(orgType: string | undefined): number {
  if (!orgType) return 0
  return EXPERTISE_MAX_BY_ORG_TYPE[orgType] ?? 0
}

// Convenience predicate — a "true" return means the expertise section
// should render at all for this org type. Used to hide the section
// entirely on enterprise orgs without scattering ad-hoc checks.
export function orgTypeSupportsExpertise(orgType: string | undefined): boolean {
  return getMaxExpertiseForOrgType(orgType) > 0
}
