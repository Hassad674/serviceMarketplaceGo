"use client"

import { useTranslations } from "next-intl"
import { Users } from "lucide-react"
import { useSearchProfiles } from "../hooks/use-search"
import { ProviderCard } from "./provider-card"
import type { SearchType } from "../api/search-api"

const TYPE_TITLES: Record<SearchType, string> = {
  freelancer: "findFreelancers",
  agency: "findAgencies",
  referrer: "findReferrers",
}

interface SearchPageProps {
  type: SearchType
}

export function SearchPage({ type }: SearchPageProps) {
  const t = useTranslations("search")
  const {
    data,
    isLoading,
    error,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
  } = useSearchProfiles(type)

  const profiles = data?.pages.flatMap((page) => page.data) ?? []

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
          {t(TYPE_TITLES[type])}
        </h1>
      </div>

      {/* Loading skeleton */}
      {isLoading && <SearchSkeleton />}

      {/* Error */}
      {error && (
        <div className="rounded-xl border border-red-200 bg-red-50 dark:border-red-500/20 dark:bg-red-500/10 p-6 text-center">
          <p className="text-sm text-red-600 dark:text-red-400">
            {t("errorLoading")}
          </p>
        </div>
      )}

      {/* Results grid */}
      {profiles.length > 0 && (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {profiles.map((profile) => (
            <ProviderCard key={profile.user_id} profile={profile} type={type} />
          ))}
        </div>
      )}

      {/* Load more button */}
      {hasNextPage && (
        <div className="flex justify-center pt-4">
          <button
            onClick={() => fetchNextPage()}
            disabled={isFetchingNextPage}
            className="rounded-lg bg-rose-500 px-6 py-2.5 text-sm font-medium text-white transition-all duration-200 ease-out hover:bg-rose-600 hover:shadow-glow active:scale-[0.98] disabled:opacity-50"
          >
            {isFetchingNextPage ? t("loading") : t("loadMore")}
          </button>
        </div>
      )}

      {/* Empty state */}
      {!isLoading && profiles.length === 0 && !error && (
        <div className="rounded-xl border border-dashed border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-900 p-12 text-center">
          <Users className="mx-auto h-10 w-10 text-gray-300 dark:text-gray-600" />
          <p className="mt-3 text-sm font-medium text-gray-500 dark:text-gray-400">
            {t("noResults")}
          </p>
          <p className="mt-1 text-xs text-gray-400 dark:text-gray-500">
            {t("noResultsDesc")}
          </p>
        </div>
      )}
    </div>
  )
}

function SearchSkeleton() {
  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
      {Array.from({ length: 6 }).map((_, i) => (
        <div
          key={i}
          className="rounded-xl border border-gray-100 dark:border-gray-800 bg-white dark:bg-gray-900 p-4 shadow-sm"
        >
          <div className="flex items-start gap-4">
            <div className="h-12 w-12 shrink-0 animate-pulse rounded-full bg-gray-200 dark:bg-gray-700" />
            <div className="flex-1 space-y-2">
              <div className="h-4 w-2/3 animate-pulse rounded bg-gray-200 dark:bg-gray-700" />
              <div className="h-3 w-1/2 animate-pulse rounded bg-gray-200 dark:bg-gray-700" />
            </div>
          </div>
        </div>
      ))}
    </div>
  )
}
