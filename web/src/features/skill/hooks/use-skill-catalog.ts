"use client"

import { useQuery } from "@tanstack/react-query"
import { fetchCatalog } from "../api/skill-api"
import { SKILLS_QUERY_KEY } from "../constants"

// Fetches the catalog of skills belonging to one expertise domain.
// Disabled when `expertiseKey` is empty so the "collapsed panel" case
// never fires a wasted request. Catalog data is highly cacheable
// (changes only when admins add/remove curated skills), hence the
// long stale time.
export function useSkillCatalog(expertiseKey: string, enabled = true) {
  return useQuery({
    queryKey: SKILLS_QUERY_KEY.catalog(expertiseKey),
    queryFn: () => fetchCatalog(expertiseKey),
    enabled: enabled && expertiseKey.length > 0,
    staleTime: 10 * 60 * 1000,
  })
}
