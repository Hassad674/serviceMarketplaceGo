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
 * The hook returns a typed result struct with documents, facet
 * counts, highlights, and the optional did-you-mean string. The
 * page composes these into the existing SearchPageLayout.
 */

import { useMemo } from "react"
import { useQuery } from "@tanstack/react-query"
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
  /** 1-indexed page number. */
  page: number
  /** Page size; capped at 60 by the backend service. */
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
  page: number
  perPage: number
  searchTimeMs: number
  correctedQuery: string | null
  isLoading: boolean
  isFetching: boolean
  error: Error | null
  refetch: () => void
}

/** EMPTY_RESULT is the placeholder returned while the key is loading. */
const EMPTY_RESULT: Omit<UseSearchResult, "isLoading" | "isFetching" | "error" | "refetch"> = {
  documents: [],
  highlights: [],
  facetCounts: {},
  found: 0,
  outOf: 0,
  page: 1,
  perPage: 20,
  searchTimeMs: 0,
  correctedQuery: null,
}

/**
 * useSearch is the top-level hook for the Typesense-backed listing
 * pages. It auto-fetches the scoped key, instantiates the client,
 * and runs the search whenever any input changes.
 *
 * Cache key includes every input so the underlying TanStack cache
 * invalidates correctly on filter / sort / pagination changes.
 */
export function useSearch(input: UseSearchInput): UseSearchResult {
  const { key, isLoading: keyLoading, error: keyError } = useSearchKey(input.persona)
  const filterBy = useMemo(() => buildFilterBy(input.filters), [input.filters])
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
        input.page,
        input.perPage ?? 20,
      ] as const,
    [input.persona, input.query, filterBy, input.sortBy, input.page, input.perPage],
  )

  const query = useQuery({
    queryKey,
    enabled,
    queryFn: async ({ signal }) => {
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
          page: input.page,
          per_page: input.perPage ?? 20,
          exclude_fields: "embedding",
          highlight_fields: "display_name,title,skills_text",
          highlight_full_fields: "display_name,title",
          num_typos: DEFAULT_NUM_TYPOS,
          max_facet_values: 40,
        },
        signal,
      )
    },
    staleTime: 30_000,
    retry: 1,
  })

  const parsed = useMemo(() => {
    if (!query.data) {
      return EMPTY_RESULT
    }
    return parseTypesenseResponse(query.data)
  }, [query.data])

  return {
    ...parsed,
    isLoading: keyLoading || query.isLoading,
    isFetching: query.isFetching,
    error: keyError ?? ((query.error as Error | null) ?? null),
    refetch: () => {
      void query.refetch()
    },
  }
}

/** parseTypesenseResponse normalises the raw response into UseSearchResult. */
function parseTypesenseResponse(
  resp: TypesenseSearchResponse,
): Omit<UseSearchResult, "isLoading" | "isFetching" | "error" | "refetch"> {
  const documents = resp.hits.map((h) => h.document)
  const highlights = resp.hits.map((h) => collectHighlights(h.highlights))
  const facetCounts: Record<string, Record<string, number>> = {}
  for (const facet of resp.facet_counts ?? []) {
    const bucket: Record<string, number> = {}
    for (const c of facet.counts) bucket[c.value] = c.count
    facetCounts[facet.field_name] = bucket
  }
  const corrected = pickCorrectedQuery(resp)
  return {
    documents,
    highlights,
    facetCounts,
    found: resp.found,
    outOf: resp.out_of,
    page: resp.page,
    perPage: resp.per_page ?? 20,
    searchTimeMs: resp.search_time_ms,
    correctedQuery: corrected,
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
