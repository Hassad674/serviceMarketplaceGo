"use client"

import { useQuery } from "@tanstack/react-query"
import { useAuth } from "@/shared/hooks/use-auth"
import { searchProfiles, type SearchType } from "../api/search-api"

const SEARCH_QUERY_KEY = "search-profiles"

export function useSearchProfiles(type: SearchType) {
  const { accessToken } = useAuth()

  return useQuery({
    queryKey: [SEARCH_QUERY_KEY, type],
    queryFn: () => searchProfiles(type, accessToken ?? undefined),
  })
}
