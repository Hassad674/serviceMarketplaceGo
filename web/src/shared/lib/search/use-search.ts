"use client"

import type { Get } from "@/shared/lib/api-paths"

/**
 * use-search.ts is the TanStack Query hook the listing pages use
 * to query the search engine. Phase 3 flips the fetch strategy:
 * instead of querying Typesense directly from the browser, the
 * hook now calls the backend proxy `/api/v1/search` — which runs
 * the hybrid query (BM25 + vector embedding) and captures the
 * query into search_queries for analytics. The backend also mints
 * the opaque cursor so the frontend never has to know about
 * Typesense pagination semantics.
 *
 * The hook composes:
 *
 *   - useInfiniteQuery with a string cursor (base64 JSON from the
 *     backend) so pagination is opaque.
 *   - buildFilterBy() — shared with the backend via the parity
 *     tests.
 *   - searchId — a deterministic hash returned by the backend for
 *     every bucketed query. Forwarded to the click-tracking hook
 *     via the second return value.
 *
 * The TypesenseSearchClient is still available for any consumer
 * that wants to bypass the proxy (not currently used but kept for
 * the eventual "logged-out anonymous search" path where hitting
 * the backend auth barrier would block the request).
 */

import { useMemo } from "react"
import { useInfiniteQuery } from "@tanstack/react-query"
import {
  type RawSearchDocument,
  type SearchDocumentPersona,
  type TypesenseHighlight,
} from "./typesense-client"
import { buildFilterBy, type SearchFilterInput } from "./build-filter-by"
import { apiClient } from "../api-client"
import type { BackendSearchPage } from "./search-types"

// Re-export so existing consumers (search-page.tsx) keep their
// import paths stable.
export type { BackendSearchPage }

/** SEARCH_COLLECTION is the alias every persona-scoped key targets. */
export const SEARCH_COLLECTION = "marketplace_actors"

/** DEFAULT_QUERY_BY mirrors the backend query_by string. */
export const DEFAULT_QUERY_BY = "display_name,title,skills_text,city"

/** DEFAULT_NUM_TYPOS matches the backend per-field typo budget. */
export const DEFAULT_NUM_TYPOS = "2,2,1,1"

/**
 * DEFAULT_SORT_BY matches the backend's three-field sort_by.
 * Typesense 28.0 rejects `_vector_distance` in sort_by unless a
 * `vector_query` is active, so the default variant keeps
 * `availability_priority`. The hybrid variant (with vector
 * distance) is emitted server-side — the backend picks the right
 * one based on whether the query was embedded.
 */
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
  /**
   * Optional first page seed used by RSC (PERF-W-02). When the page
   * is rendered as a Server Component, the listing pre-fetches the
   * first 20 documents and hands them to the client `SearchPage` as
   * `initialFirstPage`. The hook seeds TanStack Query so the first
   * paint already has cards — no client refetch unless the user
   * changes the query or filters.
   */
  initialFirstPage?: BackendSearchPage
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
  searchId: string | null
  isLoading: boolean
  isFetching: boolean
  isFetchingMore: boolean
  hasMore: boolean
  loadMore: () => void
  error: Error | null
  refetch: () => void
}

// BackendSearchPage moved to ./search-types.ts so the RSC fetcher can
// import the type without pulling the `"use client"` hook module into
// the server bundle. Re-exported from this file (above) for backward
// compatibility.

/**
 * useSearch is the top-level hook for the Typesense-backed listing
 * pages. Calls the backend proxy `/api/v1/search` which handles
 * embedding, hybrid query, and analytics capture server-side.
 * Pagination is cursor-based (opaque base64 from the server).
 */
export function useSearch(input: UseSearchInput): UseSearchResult {
  const filterBy = useMemo(() => buildFilterBy(input.filters), [input.filters])
  const perPage = input.perPage ?? DEFAULT_PER_PAGE
  const enabled = (input.enabled ?? true) && input.persona !== null

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

  // initialData seeds the first page with server-rendered results
  // (PERF-W-02). It only applies when query is empty and no filters
  // are set — i.e. the listing's default state, which matches what
  // the RSC fetched server-side. Once the user types or filters, the
  // seed is ignored and a fresh fetch is triggered automatically by
  // TanStack Query (the queryKey changes).
  const isDefaultState = input.query.trim() === "" && filterBy === ""
  const initialData =
    isDefaultState && input.initialFirstPage
      ? {
          pages: [input.initialFirstPage],
          pageParams: [""],
        }
      : undefined

  const query = useInfiniteQuery({
    queryKey,
    enabled,
    initialPageParam: "" as string,
    initialData,
    queryFn: async ({ pageParam, signal }) => {
      if (!input.persona) {
        throw new Error("useSearch: persona is required")
      }
      return fetchSearch(
        {
          persona: input.persona,
          query: input.query,
          filterBy,
          sortBy: input.sortBy ?? "",
          perPage,
          cursor: pageParam,
        },
        signal,
      )
    },
    getNextPageParam: (lastPage) =>
      lastPage.has_more && lastPage.next_cursor ? lastPage.next_cursor : undefined,
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
    isLoading: query.isLoading,
    isFetching: query.isFetching,
    isFetchingMore: query.isFetchingNextPage,
    hasMore: Boolean(query.hasNextPage),
    loadMore: () => {
      if (query.hasNextPage && !query.isFetchingNextPage) {
        void query.fetchNextPage()
      }
    },
    error: (query.error as Error | null) ?? null,
    refetch: () => {
      void query.refetch()
    },
  }
}

interface FetchSearchInput {
  persona: SearchDocumentPersona
  query: string
  filterBy: string
  sortBy: string
  perPage: number
  cursor: string
}

/**
 * fetchSearch calls the backend proxy. Exposed (not exported) so
 * the `useSearch` hook stays linear. The backend is responsible
 * for running the hybrid query, stripping the embedding from the
 * response, and attaching the next_cursor + has_more metadata.
 */
async function fetchSearch(
  input: FetchSearchInput,
  signal?: AbortSignal,
): Promise<BackendSearchPage> {
  const params = new URLSearchParams()
  params.set("persona", input.persona)
  if (input.query.trim() !== "") {
    params.set("q", input.query.trim())
  }
  if (input.sortBy) params.set("sort_by", input.sortBy)
  if (input.cursor) params.set("cursor", input.cursor)
  params.set("per_page", String(input.perPage))
  // The backend exposes each filter as a dedicated query param so
  // we unpack the filterBy string here. We deliberately do NOT send
  // the raw filter_by — the backend rebuilds it from typed params
  // so the scoped-persona invariant is enforced server-side.
  appendFilterParams(params, input.filterBy)

  const res = await apiClient<Get<"/api/v1/search"> & BackendSearchPage>(
    `/api/v1/search?${params.toString()}`,
    { method: "GET", signal },
  )
  return res
}

/**
 * appendFilterParams parses the filter_by string into the
 * individual query params the backend handler expects. The parser
 * is intentionally conservative — unknown clauses are dropped.
 */
function appendFilterParams(params: URLSearchParams, filterBy: string): void {
  if (!filterBy) return
  // Split on top-level `&&` — good enough for the small DSL the
  // builder emits (no nested groups today).
  const clauses = filterBy.split("&&").map((c) => c.trim()).filter(Boolean)
  for (const clause of clauses) {
    const matched = FILTER_PATTERNS.find((p) => p.regex.test(clause))
    if (!matched) continue
    const m = matched.regex.exec(clause)
    if (!m) continue
    matched.apply(params, m)
  }
}

interface FilterPattern {
  regex: RegExp
  apply: (params: URLSearchParams, match: RegExpExecArray) => void
}

const FILTER_PATTERNS: FilterPattern[] = [
  {
    regex: /^availability_status:\[([^\]]+)\]$/,
    apply: (p, m) => p.set("availability", m[1]),
  },
  {
    regex: /^pricing_min_amount:>=(\d+)$/,
    apply: (p, m) => p.set("pricing_min", m[1]),
  },
  {
    regex: /^pricing_max_amount:<=(\d+)$/,
    apply: (p, m) => p.set("pricing_max", m[1]),
  },
  {
    regex: /^city:"?([^"]+)"?$/,
    apply: (p, m) => p.set("city", m[1]),
  },
  {
    regex: /^country_code:([a-zA-Z]{2})$/,
    apply: (p, m) => p.set("country", m[1]),
  },
  {
    regex: /^languages_professional:\[([^\]]+)\]$/,
    apply: (p, m) => p.set("languages", m[1]),
  },
  {
    regex: /^expertise_domains:\[([^\]]+)\]$/,
    apply: (p, m) => p.set("expertise", m[1]),
  },
  {
    regex: /^skills:\[([^\]]+)\]$/,
    apply: (p, m) => p.set("skills", m[1]),
  },
  {
    regex: /^rating_average:>=([0-9.]+)$/,
    apply: (p, m) => p.set("rating_min", m[1]),
  },
  {
    regex: /^work_mode:\[([^\]]+)\]$/,
    apply: (p, m) => p.set("work_mode", m[1]),
  },
  {
    regex: /^is_verified:(true|false)$/,
    apply: (p, m) => p.set("verified", m[1]),
  },
  {
    regex: /^is_top_rated:(true|false)$/,
    apply: (p, m) => p.set("top_rated", m[1]),
  },
  {
    regex: /^pricing_negotiable:(true|false)$/,
    apply: (p, m) => p.set("negotiable", m[1]),
  },
]

/**
 * accumulatePages flattens pages into a single result view. Found /
 * facets / corrected-query come from the LATEST page so the counter
 * reflects the current filter set.
 */
function accumulatePages(
  pages: BackendSearchPage[],
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
      searchId: null,
    }
  }
  const last = pages[pages.length - 1]
  const documents: RawSearchDocument[] = []
  const highlights: Record<string, string>[] = []
  for (const p of pages) {
    for (const doc of p.documents) documents.push(doc)
    for (const h of p.highlights) highlights.push(h)
  }
  return {
    documents,
    highlights,
    facetCounts: last.facet_counts ?? {},
    found: last.found,
    outOf: last.out_of,
    searchTimeMs: last.search_time_ms,
    correctedQuery: last.corrected_query && last.corrected_query.length > 0 ? last.corrected_query : null,
    searchId: last.search_id || null,
  }
}

/**
 * extractHighlights is kept exported so any legacy caller that
 * bypasses the backend proxy can still collapse Typesense's raw
 * array-of-highlights into the flat map shape the cards expect.
 * Currently unused by the hook itself.
 */
export function extractHighlights(in_: TypesenseHighlight[]): Record<string, string> {
  const out: Record<string, string> = {}
  for (const h of in_) {
    if (out[h.field] !== undefined) continue
    out[h.field] = h.snippet
  }
  return out
}
