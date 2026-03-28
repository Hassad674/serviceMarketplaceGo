"use client"

import { useInfiniteQuery } from "@tanstack/react-query"
import { listNotifications } from "../api/notification-api"

export const NOTIFICATIONS_QUERY_KEY = ["notifications"]

export function useNotifications() {
  return useInfiniteQuery({
    queryKey: NOTIFICATIONS_QUERY_KEY,
    queryFn: ({ pageParam }) => listNotifications(pageParam),
    getNextPageParam: (lastPage) => lastPage.has_more ? lastPage.next_cursor : undefined,
    initialPageParam: undefined as string | undefined,
  })
}
