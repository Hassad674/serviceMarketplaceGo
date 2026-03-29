"use client"

import { useInfiniteQuery } from "@tanstack/react-query"
import { listNotifications } from "../api/notification-api"
import { useCurrentUserId } from "@/shared/hooks/use-current-user-id"

export function notificationsQueryKey(uid: string | undefined) {
  return ["user", uid, "notifications"] as const
}

/** @deprecated Use notificationsQueryKey(uid) instead */
export const NOTIFICATIONS_QUERY_KEY = ["notifications"]

export function useNotifications() {
  const uid = useCurrentUserId()

  return useInfiniteQuery({
    queryKey: notificationsQueryKey(uid),
    queryFn: ({ pageParam }) => listNotifications(pageParam),
    getNextPageParam: (lastPage) => lastPage.has_more ? lastPage.next_cursor : undefined,
    initialPageParam: undefined as string | undefined,
  })
}
