"use client"

import { useQuery } from "@tanstack/react-query"
import { searchProfiles, type SearchType } from "../api/search-api"

const SEARCH_QUERY_KEY = "search-profiles"

export function useSearchProfiles(type: SearchType) {
  return useQuery({
    queryKey: [SEARCH_QUERY_KEY, type],
    queryFn: () => searchProfiles(type),
  })
}
