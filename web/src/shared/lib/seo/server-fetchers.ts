/**
 * server-fetchers.ts — server-only SEO fetchers used by the public
 * profile and listing pages.
 *
 * These fetchers do NOT forward cookies — every endpoint they hit is
 * publicly readable. They return null on any error so a transient
 * backend hiccup never tanks the route render or the crawl budget.
 *
 * ISR is enabled via `next.revalidate` so re-rendering the page on a
 * subsequent request is cheap (cached HTML). The 120s window is the
 * same one used by the existing profile-metadata fetchers.
 */

import { API_BASE_URL } from "@/shared/lib/api-client"
import type { Review, AverageRating } from "@/shared/types/review"
import type { RawSearchDocument } from "@/shared/lib/search/typesense-client"
import type { SearchType } from "@/shared/lib/search/search-api"

const REVALIDATE_SECONDS = 120

function apiBase(): string {
  return API_BASE_URL || "http://localhost:8080"
}

/**
 * fetchPublicReviews fetches the most recent N reviews for an
 * organization. Reviews come back already filtered to the published
 * subset (the backend does not return pending double-blind reviews).
 */
export async function fetchPublicReviews(
  orgId: string,
  limit = 5,
): Promise<Review[] | null> {
  try {
    const url = `${apiBase()}/api/v1/reviews/org/${orgId}?limit=${limit}`
    const res = await fetch(url, { next: { revalidate: REVALIDATE_SECONDS } })
    if (!res.ok) return null
    const json = (await res.json()) as { data?: Review[] }
    return Array.isArray(json.data) ? json.data : null
  } catch {
    return null
  }
}

/**
 * fetchPublicAverageRating returns the org's published-review
 * aggregate. Returns null on any failure path so the caller can omit
 * the rating from JSON-LD without crashing.
 */
export async function fetchPublicAverageRating(
  orgId: string,
): Promise<AverageRating | null> {
  try {
    const url = `${apiBase()}/api/v1/reviews/average/${orgId}`
    const res = await fetch(url, { next: { revalidate: REVALIDATE_SECONDS } })
    if (!res.ok) return null
    const json = (await res.json()) as { data?: AverageRating }
    if (!json.data) return null
    return json.data
  } catch {
    return null
  }
}

/**
 * pickRelatedProfiles selects up to `limit` related profiles for the
 * "Profils similaires" section, prioritising:
 *   1. Same primary expertise domain.
 *   2. Same city.
 *
 * Falls back to general high-rated docs when neither match exists so
 * the section never renders empty (which would be both a UX miss and
 * an SEO opportunity wasted).
 */
export interface PickRelatedInput {
  candidates: RawSearchDocument[]
  excludeOrgId: string
  primaryExpertise: string | undefined
  city: string | undefined
  limit: number
}

export function pickRelatedProfiles(
  input: PickRelatedInput,
): RawSearchDocument[] {
  const { candidates, excludeOrgId, primaryExpertise, city, limit } = input
  if (candidates.length === 0) return []

  const filtered = candidates.filter((doc) => {
    const docOrgId = doc.organization_id ?? doc.id.split(":")[0] ?? doc.id
    return docOrgId !== excludeOrgId
  })

  // Score docs to keep the deterministic ordering Google likes.
  const scored = filtered.map((doc) => {
    let score = 0
    if (primaryExpertise && doc.expertise_domains?.includes(primaryExpertise)) {
      score += 10
    }
    if (city && doc.city === city) {
      score += 4
    }
    if (doc.is_top_rated) score += 1
    if (doc.is_featured) score += 1
    score += Math.min(doc.rating_count, 5) * 0.1
    return { doc, score }
  })

  scored.sort((a, b) => {
    if (b.score !== a.score) return b.score - a.score
    return b.doc.rating_average - a.doc.rating_average
  })

  return scored.slice(0, limit).map((entry) => entry.doc)
}

/**
 * fetchRelatedProfiles wraps `fetchListingFirstPage` with the persona
 * mapping and post-filter logic. It returns up to `limit` documents
 * suitable for the "Related profiles" cross-link section.
 *
 * The fetcher is intentionally thin — it lives next to the other SEO
 * helpers so the page-level imports stay focused.
 */
export interface FetchRelatedInput {
  type: SearchType
  excludeOrgId: string
  primaryExpertise: string | undefined
  city: string | undefined
  limit?: number
}

export async function fetchRelatedProfiles(
  input: FetchRelatedInput,
): Promise<RawSearchDocument[]> {
  const limit = input.limit ?? 6
  // Lazy import to keep the module's static import graph minimal —
  // search-server lives in a feature folder; we don't want every SEO
  // consumer to drag it in transitively.
  const { fetchListingFirstPage } = await import(
    "@/features/provider/api/search-server"
  )
  const page = await fetchListingFirstPage(input.type)
  if (!page) return []
  return pickRelatedProfiles({
    candidates: page.documents,
    excludeOrgId: input.excludeOrgId,
    primaryExpertise: input.primaryExpertise,
    city: input.city,
    limit,
  })
}
