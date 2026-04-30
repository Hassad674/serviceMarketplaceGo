"use client"

import { Suspense } from "react"
import { useSearchParams } from "next/navigation"
import { SearchPage } from "@/features/provider/components/search-page"
import type { SearchType } from "@/shared/lib/search/search-api"

const VALID_TYPES: SearchType[] = ["freelancer", "agency", "referrer"]

function SearchContent() {
  const searchParams = useSearchParams()
  const typeParam = searchParams.get("type")
  const type: SearchType = VALID_TYPES.includes(typeParam as SearchType)
    ? (typeParam as SearchType)
    : "freelancer"

  return <SearchPage type={type} />
}

export default function SearchRoutePage() {
  return (
    <Suspense fallback={<SearchPageSkeleton />}>
      <SearchContent />
    </Suspense>
  )
}

function SearchPageSkeleton() {
  return (
    <div className="space-y-6">
      <div className="h-8 w-48 animate-pulse rounded bg-gray-200 dark:bg-gray-700" />
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
    </div>
  )
}
