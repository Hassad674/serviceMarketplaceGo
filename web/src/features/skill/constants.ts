// Per-org-type maximum number of profile skills. Mirrors backend
// validation — if these diverge, the UI still caps selection locally
// and the backend rejects oversize payloads with a 400 error.
// `enterprise` has no skills section at all, hence 0.
//
// Kept as a local constant (duplicated from the backend instead of
// fetched) so the skill feature has zero runtime dependencies on
// other features. If the backend grows new org types we just add a
// line here.
export const SKILLS_MAX_BY_ORG_TYPE: Record<string, number> = {
  agency: 40,
  provider_personal: 25,
  enterprise: 0,
}

export function getMaxSkillsForOrgType(orgType: string | undefined): number {
  if (!orgType) return 0
  return SKILLS_MAX_BY_ORG_TYPE[orgType] ?? 0
}

// Convenience predicate — `true` means the skills section should
// render at all for this org type. Hides the feature entirely for
// enterprises without scattering ad-hoc checks across components.
export function orgTypeSupportsSkills(orgType: string | undefined): boolean {
  return getMaxSkillsForOrgType(orgType) > 0
}

// Centralised query keys so hooks and tests reference the same
// canonical shape. Using `as const` gives TanStack Query precise
// tuple types for cache invalidation.
export const SKILLS_QUERY_KEY = {
  profile: ["skills", "profile"] as const,
  catalog: (expertise: string) => ["skills", "catalog", expertise] as const,
  autocomplete: (query: string) => ["skills", "autocomplete", query] as const,
}

// Debounce delay for the autocomplete search input. 200ms is the
// sweet spot where fast typers are not spammed with requests while
// the UI still feels immediate.
export const SKILL_AUTOCOMPLETE_DEBOUNCE_MS = 200

// Maximum number of popular skills surfaced in the "popular in your
// domains" row inside the editor modal.
export const POPULAR_SKILLS_LIMIT = 8
