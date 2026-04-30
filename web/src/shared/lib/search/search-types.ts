/**
 * search-types.ts holds the typed envelopes shared between the
 * client search hook (`use-search.ts`) and the RSC server fetcher
 * (`provider/api/search-server.ts`). Living in its own file lets
 * the server bundle import the types without dragging the
 * `"use client"` hook module into the RSC graph.
 */

import type { RawSearchDocument } from "./typesense-client"

/**
 * BackendSearchPage is the wire shape returned by the public
 * `/api/v1/search` proxy. The frontend never inspects every field —
 * the unused ones (highlights / facet_counts / out_of) are kept
 * because they round-trip through TanStack Query's pageParam cache
 * when the user paginates.
 */
export interface BackendSearchPage {
  search_id: string
  documents: RawSearchDocument[]
  highlights: Record<string, string>[]
  facet_counts: Record<string, Record<string, number>>
  found: number
  out_of: number
  page: number
  per_page: number
  search_time_ms: number
  corrected_query?: string
  next_cursor?: string
  has_more: boolean
}
