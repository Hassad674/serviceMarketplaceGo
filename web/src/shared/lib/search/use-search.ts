"use client"

/**
 * use-search.ts is the TanStack Query hook the listing pages use
 * to query Typesense directly. It composes:
 *
 *   - useSearchKey()         — fetches + caches the scoped key
 *   - TypesenseSearchClient  — minimal HTTP wrapper
 *   - buildFilterBy()        — translates filter inputs into
 *                              Typesense filter_by syntax
 *
 * Uses `useInfiniteQuery` under the hood so pagination is handled
 * natively by TanStack Query — each page is a separate fetch that
 * gets appended to the accumulated result set. The listing page
 * only has to call `loadMore()` when the user scrolls past the
 * fold.
 */

import { useMemo } from "react"
import { useInfiniteQuery } from "@tanstack/react-query"
import {
  TypesenseSearchClient,
  type RawSearchDocument,
  type SearchDocumentPersona,
  type TypesenseHighlight,
  type TypesenseSearchResponse,
} from "./typesense-client"
import { useSearchKey } from "./use-search-key"
import { buildFilterBy, type SearchFilterInput } from "./build-filter-by"

/** SEARCH_COLLECTION is the alias every persona-scoped key targets. */
export const SEARCH_COLLECTION = "marketplace_actors"

/** DEFAULT_QUERY_BY mirrors the backend query_by string. */
export const DEFAULT_QUERY_BY = "display_name,title,skills_text,city"

/** DEFAULT_NUM_TYPOS matches the backend per-field typo budget. */
export const DEFAULT_NUM_TYPOS = "2,2,1,1"

/** DEFAULT_SORT_BY matches the backend's three-field sort_by. */
export const DEFAULT_SORT_BY =
  "_text_match(buckets:10):desc,availability_priority:desc,rating_score:desc"

/** DEFAULT_FACET_BY mirrors the fields the sidebar exposes counts for. */
export const DEFAULT_FACET_BY =
  "availability_status,city,country_code,languages_professional," +
  "expertise_domains,skills,work_mode,is_verified,is_top_rated,pricing_currency"

/** DEFAULT_PER_PAGE is the page size used by the listing grid. */
export const DEFAULT_PER_PAGE = 20

/** UseSearchInput is the per-call parameter struct. */
export interface UseSearchInput {
  /** Persona is the scoped client to use. */
  persona: SearchDocumentPersona | null
  /** Free-text query. Empty string becomes "*" (match-all). */
  query: string
  /** Filter sidebar payload. */
  filters: SearchFilterInput
  /** Override the default sort_by. Empty = use DEFAULT_SORT_BY. */
  sortBy?: string
  /** Page size. Capped at 60 by the backend service. */
  perPage?: number
  /** Disables the query when false. */
  enabled?: boolean
}

/** UseSearchResult is the shape callers consume. */
export interface UseSearchResult {
  documents: RawSearchDocument[]
  highlights: Record<string, string>[]
  facetCounts: Record<string, Record<string, number>>
  found: number
  outOf: number
  perPage: number
  searchTimeMs: number
  correctedQuery: string | null
  isLoading: boolean
  isFetching: boolean
  isFetchingMore: boolean
  hasMore: boolean
  loadMore: () => void
  error: Error | null
  refetch: () => void
}

/**
 * useSearch is the top-level hook for the Typesense-backed listing
 * pages. It auto-fetches the scoped key, instantiates the client,
 * and accumulates pages via TanStack Query's infinite-query flow.
 *
 * Cache key includes every query shape (persona, query, filters,
 * sort) so mutating any of them transparently resets the
 * accumulator back to page 1.
 */
export function useSearch(input: UseSearchInput): UseSearchResult {
  const { key, isLoading: keyLoading, error: keyError } = useSearchKey(input.persona)
  const filterBy = useMemo(() => buildFilterBy(input.filters), [input.filters])
  const perPage = input.perPage ?? DEFAULT_PER_PAGE
  const enabled = (input.enabled ?? true) && key !== null && input.persona !== null

  const queryKey = useMemo(
    () =>
      [
        "search",
        "results",
        input.persona,
        input.query,
        filterBy,
        input.sortBy ?? "",
        perPage,
      ] as const,
    [input.persona, input.query, filterBy, input.sortBy, perPage],
  )

  const query = useInfiniteQuery({
    queryKey,
    enabled,
    initialPageParam: 1,
    queryFn: async ({ pageParam, signal }) => {
      if (!key) throw new Error("useSearch: scoped key not yet available")
      const client = new TypesenseSearchClient(key.host, key.key)
      return client.search(
        SEARCH_COLLECTION,
        {
          q: input.query.trim() === "" ? "*" : input.query.trim(),
          query_by: DEFAULT_QUERY_BY,
          filter_by: filterBy || undefined,
          facet_by: DEFAULT_FACET_BY,
          sort_by: input.sortBy && input.sortBy.length > 0 ? input.sortBy : DEFAULT_SORT_BY,
          page: pageParam,
          per_page: perPage,
          exclude_fields: "embedding",
          highlight_fields: "display_name,title,skills_text",
          highlight_full_fields: "display_name,title",
          num_typos: DEFAULT_NUM_TYPOS,
          max_facet_values: 40,
        },
        signal,
      )
    },
    getNextPageParam: (lastPage) => {
      const effectivePerPage = lastPage.per_page ?? perPage
      const loaded = lastPage.page * effectivePerPage
      if (loaded >= lastPage.found) return undefined
      return lastPage.page + 1
    },
    staleTime: 30_000,
    retry: 1,
  })

  const accumulated = useMemo(
    () => accumulatePages(query.data?.pages ?? []),
    [query.data],
  )

  return {
    ...accumulated,
    perPage,
    isLoading: keyLoading || query.isLoading,
    isFetching: query.isFetching,
    isFetchingMore: query.isFetchingNextPage,
    hasMore: Boolean(query.hasNextPage),
    loadMore: () => {
      if (query.hasNextPage && !query.isFetchingNextPage) {
        void query.fetchNextPage()
      }
    },
    error: keyError ?? ((query.error as Error | null) ?? null),
    refetch: () => {
      void query.refetch()
    },
  }
}

/**
 * accumulatePages flattens pages into a single result view. Found /
 * facets / corrected-query come from the LATEST page so the counter
 * reflects the current filter set (TanStack invalidates the cache
 * whenever the queryKey changes).
 */
function accumulatePages(
  pages: TypesenseSearchResponse[],
): Omit<
  UseSearchResult,
  | "perPage"
  | "isLoading"
  | "isFetching"
  | "isFetchingMore"
  | "hasMore"
  | "loadMore"
  | "error"
  | "refetch"
> {
  if (pages.length === 0) {
    return {
      documents: [],
      highlights: [],
      facetCounts: {},
      found: 0,
      outOf: 0,
      searchTimeMs: 0,
      correctedQuery: null,
    }
  }
  const last = pages[pages.length - 1]
  const documents: RawSearchDocument[] = []
  const highlights: Record<string, string>[] = []
  for (const p of pages) {
    for (const h of p.hits) {
      documents.push(h.document)
      highlights.push(collectHighlights(h.highlights))
    }
  }
  const facetCounts: Record<string, Record<string, number>> = {}
  for (const facet of last.facet_counts ?? []) {
    const bucket: Record<string, number> = {}
    for (const c of facet.counts) bucket[c.value] = c.count
    facetCounts[facet.field_name] = bucket
  }
  return {
    documents,
    highlights,
    facetCounts,
    found: last.found,
    outOf: last.out_of,
    searchTimeMs: last.search_time_ms,
    correctedQuery: pickCorrectedQuery(last),
  }
}

function collectHighlights(in_: TypesenseHighlight[]): Record<string, string> {
  const out: Record<string, string> = {}
  for (const h of in_) {
    if (out[h.field] !== undefined) continue
    out[h.field] = h.snippet
  }
  return out
}

function pickCorrectedQuery(resp: TypesenseSearchResponse): string | null {
  if (resp.corrected_query && resp.corrected_query.length > 0) return resp.corrected_query
  const first = resp.request_params.first_q
  const ran = resp.request_params.q
  if (first && ran && first !== ran) return ran
  return null
}
