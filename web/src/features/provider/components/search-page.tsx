"use client"

import { useCallback, useState } from "react"
import { useTranslations } from "next-intl"
import {
  SearchPageLayout,
  type SortKey,
} from "@/shared/components/search/search-page-layout"
import { DidYouMeanBanner } from "@/shared/components/search/did-you-mean-banner"
import {
  EMPTY_SEARCH_FILTERS,
  type SearchFilters,
} from "@/shared/components/search/search-filters"
import type {
  SearchDocumentPersona,
} from "@/shared/lib/search/search-document"
import { useSearch, type BackendSearchPage } from "@/shared/lib/search/use-search"
import { trackSearchClick } from "@/shared/lib/search/track-click"
import { fromTypesenseDocument } from "@/shared/lib/search/typesense-document-adapter"
import type { SearchType } from "@/shared/lib/search/search-api"

/**
 * SearchPage is the public listings root for freelancers, agencies,
 * and referrers. It wires the shared SearchPageLayout to the
 * Typesense-backed useSearch hook — there is no longer a SQL
 * fallback (the 30-day grace period ended with phase 4).
 *
 * Behaviour:
 *  - SUBMIT-ONLY query input — typing never triggers a fetch; the
 *    backend is hit only when the user presses Enter or clicks the
 *    magnifier icon. Drives perceived performance and avoids burning
 *    Typesense credits for half-typed queries.
 *  - filter sidebar maps into the Typesense filter_by builder, but
 *    edits stay local until the user clicks "Apply" — at which point
 *    the draft is promoted to applied and a single fetch fires.
 *  - did-you-mean banner over the results grid when Typesense
 *    returns a `corrected_query`
 *  - click tracking beacon fired on card select for CTR analytics
 */

const TYPE_TITLES: Record<SearchType, string> = {
  freelancer: "findFreelancers",
  agency: "findAgencies",
  enterprise: "findFreelancers", // unused — defensive fallback
  referrer: "findReferrers",
}

const TYPE_TO_PERSONA: Record<SearchType, SearchDocumentPersona> = {
  freelancer: "freelance",
  agency: "agency",
  referrer: "referrer",
  // Enterprise is not a discoverable persona in the redesign — mapping
  // to freelance keeps the SearchDocument adapter happy; the feature
  // never actually serves an enterprise listing through this component.
  enterprise: "freelance",
}

interface SearchPageProps {
  type: SearchType
  /**
   * Optional first-page seed produced by the RSC pass (PERF-W-02).
   * When present, the hook short-circuits the initial network round
   * trip and renders the cards from the SSR HTML — Googlebot sees
   * the listing without waiting for hydration, and users see no
   * "loading" flash on first paint.
   */
  initialFirstPage?: BackendSearchPage
  /**
   * Optional initial query string. The public listing routes
   * (/freelancers, /agencies, /referrers) read `?q=` from the URL
   * (set by the landing page search bar) and forward it here so the
   * results panel prefills the input AND fires a fetch on mount.
   * Empty string keeps the unscoped catalog behaviour.
   */
  initialQuery?: string
}

export function SearchPage({
  type,
  initialFirstPage,
  initialQuery = "",
}: SearchPageProps) {
  const t = useTranslations("search")
  // queryDraft owns the live input value; appliedQuery is what we
  // feed the search hook. Decoupling them is the whole point of
  // submit-only — the user types freely without burning network
  // requests until Enter / magnifier click.
  //
  // initialQuery seeds BOTH so the URL `?q=foo` produces a real
  // results panel (not just a pre-filled empty input). A non-empty
  // initialQuery also invalidates the RSC seed because that seed is
  // unscoped — let the hook refetch with the actual query.
  const [queryDraft, setQueryDraft] = useState(initialQuery)
  const [appliedQuery, setAppliedQuery] = useState(initialQuery)
  // Same draft/applied split for filters: edits stay local until
  // the user clicks "Apply" in the sidebar, mirroring the input UX.
  const [filtersDraft, setFiltersDraft] =
    useState<SearchFilters>(EMPTY_SEARCH_FILTERS)
  const [appliedFilters, setAppliedFilters] =
    useState<SearchFilters>(EMPTY_SEARCH_FILTERS)
  const [sort, setSort] = useState<SortKey>("relevance")
  const persona = TYPE_TO_PERSONA[type]

  const result = useSearch({
    persona,
    query: appliedQuery,
    filters: filtersToInput(appliedFilters),
    sortBy: sortKeyToTypesense(sort),
    perPage: 20,
    // The RSC seed is fetched without a query; pass it through only
    // when the page boots without an initial query so the seed is a
    // valid match. Otherwise let the hook refetch with the user's
    // query — the seed would mislead the results panel.
    initialFirstPage: appliedQuery ? undefined : initialFirstPage,
  })

  const handleSelect = useCallback(
    (docID: string, position: number) => {
      if (!result.searchId) return
      trackSearchClick(result.searchId, docID, position)
    },
    [result.searchId],
  )

  // The layout calls this on Enter / magnifier-click. Drafts are
  // already in the parent's state, so we just promote them.
  const handleQuerySubmit = useCallback((next: string) => {
    setAppliedQuery(next)
  }, [])

  const handleFiltersApply = useCallback(() => {
    setFiltersDraft((current) => {
      setAppliedFilters(current)
      return current
    })
  }, [])

  // The reset button on the sidebar swaps both the draft and the
  // applied state to EMPTY_SEARCH_FILTERS in one go — otherwise the
  // user would have to click reset + apply.
  const handleFiltersChange = useCallback((next: SearchFilters) => {
    setFiltersDraft(next)
    if (next === EMPTY_SEARCH_FILTERS) {
      setAppliedFilters(EMPTY_SEARCH_FILTERS)
    }
  }, [])

  const status: "loading" | "error" | "idle" = result.isLoading
    ? "loading"
    : result.error
      ? "error"
      : "idle"

  // Strip the embedding vector + adapt to the frozen card contract.
  const documents = result.documents.map((doc) => fromTypesenseDocument(doc))

  return (
    <div className="flex flex-col gap-4">
      {result.correctedQuery ? (
        <DidYouMeanBanner
          correctedQuery={result.correctedQuery}
          onApply={(corrected) => {
            setQueryDraft(corrected)
            setAppliedQuery(corrected)
          }}
        />
      ) : null}
      <SearchPageLayout
        title={t(TYPE_TITLES[type])}
        persona={persona}
        preMappedDocuments={documents}
        status={status}
        hasMore={result.hasMore}
        isLoadingMore={result.isFetchingMore}
        onLoadMore={result.loadMore}
        onRetry={result.refetch}
        query={queryDraft}
        onQueryChange={setQueryDraft}
        onQuerySubmit={handleQuerySubmit}
        filters={filtersDraft}
        onFiltersChange={handleFiltersChange}
        onFiltersApply={handleFiltersApply}
        sort={sort}
        onSortChange={setSort}
        totalFound={result.found}
        onSelect={handleSelect}
      />
    </div>
  )
}

// sortKeyToTypesense maps the UI's SortKey enum to a Typesense
// `sort_by` string. Capped at three fields because Typesense 28.0
// rejects longer sort chains at query time.
function sortKeyToTypesense(key: SortKey): string {
  switch (key) {
    case "rating":
      return "rating_score:desc,rating_count:desc,_text_match(buckets:10):desc"
    case "priceAsc":
      return "pricing_min_amount:asc,_text_match(buckets:10):desc,rating_score:desc"
    case "priceDesc":
      return "pricing_min_amount:desc,_text_match(buckets:10):desc,rating_score:desc"
    case "recent":
      return "last_active_at:desc,_text_match(buckets:10):desc,rating_score:desc"
    case "relevance":
    default:
      // Backend swaps availability_priority for _vector_distance when
      // hybrid search is active (user typed something). We emit the
      // BM25-friendly variant here so the backend can override safely.
      return "_text_match(buckets:10):desc,availability_priority:desc,rating_score:desc"
  }
}

/**
 * filtersToInput projects the SearchFilters state owned by the
 * sidebar into the Typesense FilterInput shape consumed by
 * `buildFilterBy`. Keep in parity with the backend's typed
 * FilterInput in internal/app/search/filter_builder.go.
 */
function filtersToInput(filters: SearchFilters) {
  const availabilityStatus =
    filters.availability === "all"
      ? undefined
      : filters.availability === "now"
        ? ["available_now"]
        : ["available_soon"]
  return {
    availabilityStatus,
    pricingMin: filters.priceMin,
    pricingMax: filters.priceMax,
    city: filters.city,
    countryCode: filters.countryCode,
    languages: filters.languages,
    expertiseDomains: filters.expertise as string[],
    skills: filters.skills,
    ratingMin: filters.minRating > 0 ? filters.minRating : undefined,
    workMode: filters.workModes,
  }
}
