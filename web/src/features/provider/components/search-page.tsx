"use client"

import { useState } from "react"
import { useTranslations } from "next-intl"
import { SearchPageLayout } from "@/shared/components/search/search-page-layout"
import type {
  SearchDocumentPersona,
} from "@/shared/lib/search/search-document"
import type { RawSearchDocumentLike } from "@/shared/lib/search/search-document-adapter"
import { useSearchProfiles } from "../hooks/use-search"
import type { SearchType } from "../api/search-api"

// SearchPage is the thin provider-feature adapter that wires the
// existing TanStack Query hook into the shared SearchPageLayout. The
// layout itself lives in shared/ so all three public listings
// (freelancers / agencies / referrers) can compose it identically —
// this component only exists because the app/search route still
// fetches profiles via the provider feature's hook.
//
// The persona prop on the layout controls both the detail-page path
// the cards link to and which pricing kind is pulled onto the card.

const TYPE_TITLES: Record<SearchType, string> = {
  freelancer: "findFreelancers",
  agency: "findAgencies",
  enterprise: "findEnterprises",
  referrer: "findReferrers",
}

const TYPE_TO_PERSONA: Record<SearchType, SearchDocumentPersona> = {
  freelancer: "freelance",
  agency: "agency",
  referrer: "referrer",
  // Enterprise is not a discoverable persona in the redesign — it
  // still maps to freelance at the contract level so the SearchDocument
  // adapter has a fallback; the feature never actually serves an
  // enterprise listing through this component.
  enterprise: "freelance",
}

interface SearchPageProps {
  type: SearchType
}

export function SearchPage({ type }: SearchPageProps) {
  const t = useTranslations("search")
  const [query, setQuery] = useState("")
  const {
    data,
    isLoading,
    error,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
    refetch,
  } = useSearchProfiles(type)

  const documents: RawSearchDocumentLike[] =
    data?.pages.flatMap((page) => page.data as RawSearchDocumentLike[]) ?? []

  const status: "loading" | "error" | "idle" = isLoading
    ? "loading"
    : error
      ? "error"
      : "idle"

  return (
    <SearchPageLayout
      title={t(TYPE_TITLES[type])}
      persona={TYPE_TO_PERSONA[type]}
      documents={documents}
      status={status}
      hasMore={Boolean(hasNextPage)}
      isLoadingMore={isFetchingNextPage}
      onLoadMore={() => fetchNextPage()}
      onRetry={() => refetch()}
      query={query}
      onQueryChange={setQuery}
    />
  )
}
