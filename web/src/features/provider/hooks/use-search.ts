"use client"

import { useInfiniteQuery } from "@tanstack/react-query"
import { searchProfiles, type SearchType } from "../api/search-api"

const SEARCH_QUERY_KEY = "search-profiles"

export function useSearchProfiles(type: SearchType) {
  return useInfiniteQuery({
    queryKey: [SEARCH_QUERY_KEY, type],
    queryFn: ({ pageParam }) => searchProfiles(type, pageParam),
    getNextPageParam: (lastPage) =>
      lastPage.has_more ? lastPage.next_cursor : undefined,
    initialPageParam: undefined as string | undefined,
    staleTime: 2 * 60 * 1000, // 2 minutes — search results change infrequently
  })
}
