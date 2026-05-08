"use client"

import { useInfiniteQuery } from "@tanstack/react-query"
import {
  listSecurityActivity,
  type SecurityActivityResponse,
} from "../api/security-api"

const PAGE_SIZE = 20

/**
 * useSecurityActivity — TanStack infinite query over
 * /api/v1/me/security/activity.
 *
 * Returns pages of auth events newest-first; the consumer flattens
 * the pages into a single list and renders a "Voir plus" CTA when
 * `hasNextPage` is true.
 *
 * staleTime = 60s so the user can switch tabs and back without
 * refetching every time; refetchOnWindowFocus = false because a
 * security log does not need real-time updates and the
 * cursor-window dance is jarring on focus changes.
 */
export function useSecurityActivity() {
  return useInfiniteQuery<SecurityActivityResponse>({
    queryKey: ["security", "activity"],
    queryFn: ({ pageParam }) =>
      listSecurityActivity({
        cursor: typeof pageParam === "string" ? pageParam : undefined,
        limit: PAGE_SIZE,
      }),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (last) => last.next_cursor || undefined,
    staleTime: 60_000,
    refetchOnWindowFocus: false,
  })
}
