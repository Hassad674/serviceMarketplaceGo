"use client"

import { useInfiniteQuery } from "@tanstack/react-query"
import {
  getReferrerReputation,
  type ReferrerReputation,
} from "../api/reputation-api"

const DEFAULT_PAGE_SIZE = 20

// useReferrerReputation reads the apporteur's reputation aggregate:
// summary rating + cursor-paginated project history. The summary
// stats come from the first page — every subsequent page only carries
// additional history entries.
//
// Keyed on orgID to stay consistent with the rest of the referrer
// profile surface; the backend translates internally because
// referrals reference users.
export function useReferrerReputation(orgId: string | undefined) {
  return useInfiniteQuery({
    queryKey: ["referrer-reputation", orgId],
    queryFn: ({ pageParam }) =>
      getReferrerReputation(orgId!, {
        cursor: pageParam,
        limit: DEFAULT_PAGE_SIZE,
      }),
    initialPageParam: "",
    getNextPageParam: (lastPage: ReferrerReputation) =>
      lastPage.has_more ? lastPage.next_cursor : undefined,
    staleTime: 2 * 60 * 1000,
    enabled: Boolean(orgId),
  })
}
