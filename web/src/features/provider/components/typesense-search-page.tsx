"use client"

import { useState } from "react"
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
import { useSearch } from "@/shared/lib/search/use-search"
import { fromTypesenseDocument } from "@/shared/lib/search/typesense-document-adapter"
import type { SearchType } from "../api/search-api"

/**
 * TypesenseSearchPage is the Typesense-backed counterpart of
 * SearchPage (the legacy SQL adapter). It is selected by SearchPage
 * when the NEXT_PUBLIC_SEARCH_ENGINE feature flag is set to
 * "typesense".
 *
 * Behaviour:
 *  - debounced 250 ms search input
 *  - filter sidebar wired into the Typesense filter_by builder
 *  - did-you-mean banner over the results grid when the cluster
 *    returns a corrected query
 *  - highlights propagated to the result cards via the layout
 *    (today the cards do not yet render <mark> tags — wiring left
 *    in place so the data is available for the next iteration)
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
  enterprise: "freelance",
}

interface TypesenseSearchPageProps {
  type: SearchType
}

export function TypesenseSearchPage({ type }: TypesenseSearchPageProps) {
  const t = useTranslations("search")
  const [query, setQuery] = useState("")
  const [filters, setFilters] = useState<SearchFilters>(EMPTY_SEARCH_FILTERS)
  const [sort, setSort] = useState<SortKey>("relevance")
  const persona = TYPE_TO_PERSONA[type]

  const result = useSearch({
    persona,
    query,
    filters: filtersToInput(filters),
    sortBy: sortKeyToTypesense(sort),
    perPage: 20,
  })

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
          onApply={(corrected) => setQuery(corrected)}
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
        query={query}
        onQueryChange={setQuery}
        filters={filters}
        onFiltersChange={setFilters}
        sort={sort}
        onSortChange={setSort}
        totalFound={result.found}
      />
    </div>
  )
}

// sortKeyToTypesense maps the UI's SortKey enum to a Typesense
// `sort_by` string. We cap at three fields because Typesense 28.0
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
      return "_text_match(buckets:10):desc,availability_priority:desc,rating_score:desc"
  }
}

/**
 * filtersToInput projects the SearchFilters state owned by the
 * sidebar into the Typesense FilterInput shape consumed by
 * `buildFilterBy`. The mapping matches the wire format on the
 * backend: the sidebar's "all" availability becomes an empty
 * `availabilityStatus` slice instead of the literal string "all".
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
