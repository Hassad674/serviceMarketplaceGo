import type { SearchDocumentPersona } from "@/shared/lib/search/search-document"

/**
 * SearchFilterVisibility describes which filter sections are shown
 * for a given search persona. The sidebar reads this map to decide
 * which sub-sections to render — keeping the logic data-driven so
 * adding a new persona never means editing every section component.
 *
 * Per product brief (2026-05):
 *   - freelance: every filter is shown.
 *   - agency:   no work-mode filter (agencies don't expose a per-engagement
 *               remote/onsite/hybrid flag the way freelances do).
 *   - referrer: no skill, no work-mode, no pricing — referrers monetise
 *               via commission %, not project price; they connect rather
 *               than execute, so technical skills don't apply.
 */
export interface SearchFilterVisibility {
  availability: boolean
  pricing: boolean
  location: boolean
  workMode: boolean
  languages: boolean
  expertise: boolean
  skills: boolean
  rating: boolean
}

const FREELANCE_VISIBILITY: SearchFilterVisibility = {
  availability: true,
  pricing: true,
  location: true,
  workMode: true,
  languages: true,
  expertise: true,
  skills: true,
  rating: true,
}

const AGENCY_VISIBILITY: SearchFilterVisibility = {
  availability: true,
  pricing: true,
  location: true,
  workMode: false,
  languages: true,
  expertise: true,
  skills: true,
  rating: true,
}

const REFERRER_VISIBILITY: SearchFilterVisibility = {
  availability: true,
  pricing: false,
  location: true,
  workMode: false,
  languages: true,
  expertise: true,
  skills: false,
  rating: true,
}

/**
 * FILTERS_BY_PERSONA is the canonical mapping from a search persona
 * to the visibility flags. Exported separately so the test suite can
 * snapshot the table without having to reach into the hook.
 */
export const FILTERS_BY_PERSONA: Record<
  SearchDocumentPersona,
  SearchFilterVisibility
> = {
  freelance: FREELANCE_VISIBILITY,
  agency: AGENCY_VISIBILITY,
  referrer: REFERRER_VISIBILITY,
}

/**
 * resolveFilterVisibility returns the visibility map for a persona.
 * Falls back to the freelance map (the most permissive) when the
 * persona is undefined — keeps legacy callers that never picked a
 * persona working without crashing.
 */
export function resolveFilterVisibility(
  persona: SearchDocumentPersona | undefined,
): SearchFilterVisibility {
  if (!persona) return FREELANCE_VISIBILITY
  return FILTERS_BY_PERSONA[persona] ?? FREELANCE_VISIBILITY
}
