/**
 * search-server.ts is the server-only fetcher used by the public
 * listing pages (`/agencies`, `/freelancers`, `/referrers`) to
 * pre-render the first page of search results in the RSC pass.
 *
 * Why server-side?
 *   - SEO: Googlebot sees real HTML cards instead of an empty
 *     `<div>` followed by hydration JS (PERF-W-02).
 *   - LCP: the first 20 cards land in the initial HTML payload,
 *     skipping a render-blocking client fetch.
 *
 * The fetcher hits the same `/api/v1/search` proxy the client uses,
 * but with no cookies (the endpoint is public). ISR (`revalidate:
 * 60`) keeps the cache warm for a minute — listings change slowly
 * enough that 60s is a safe trade-off between freshness and load.
 */

import { API_BASE_URL } from "@/shared/lib/api-client"
import type { SearchDocumentPersona } from "@/shared/lib/search/typesense-client"
import type { SearchType } from "@/shared/lib/search/search-api"
// search-types is the server-friendly types module — `use-search.ts`
// carries `"use client"` and must not be imported from RSC code,
// even type-only, to keep the import graph clean.
import type { BackendSearchPage } from "@/shared/lib/search/search-types"

// SearchServerPage is the typed envelope returned by the public
// `/api/v1/search` proxy. It is the BackendSearchPage shape — kept
// as a re-export so the page tests can import without dragging in
// the client-only useSearch hook.
export type SearchServerPage = BackendSearchPage

const TYPE_TO_PERSONA: Record<SearchType, SearchDocumentPersona> = {
  freelancer: "freelance",
  agency: "agency",
  referrer: "referrer",
  // Enterprise persona doesn't surface through public listings — this
  // mapping is defensive and never hit by the public flow.
  enterprise: "freelance",
}

/**
 * fetchListingFirstPage runs server-side and returns the first 20
 * documents for the requested public listing. Returns null on any
 * error so the page can render a graceful empty state without
 * crashing the route — listings are SEO surfaces and a 500 here
 * would tank crawl budget.
 */
// DEFAULT_SORT_BY duplicated from `shared/lib/search/use-search.ts`
// to keep this module zero-dep on client code (the import would
// pull TanStack Query into the server bundle for nothing).
const DEFAULT_SORT_BY =
  "_text_match(buckets:10):desc,availability_priority:desc,rating_score:desc"

export async function fetchListingFirstPage(
  type: SearchType,
): Promise<SearchServerPage | null> {
  try {
    const persona = TYPE_TO_PERSONA[type]
    const params = new URLSearchParams({
      persona,
      per_page: "20",
      sort_by: DEFAULT_SORT_BY,
    })
    const url = `${API_BASE_URL || "http://localhost:8080"}/api/v1/search?${params.toString()}`
    const res = await fetch(url, { next: { revalidate: 60 } })
    if (!res.ok) return null
    const json = (await res.json()) as SearchServerPage
    return json
  } catch {
    return null
  }
}
