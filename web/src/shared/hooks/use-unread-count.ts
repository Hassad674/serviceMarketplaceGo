"use client"

import { useQuery } from "@tanstack/react-query"
import { apiClient } from "@/shared/lib/api-client"
import type { Get } from "@/shared/lib/api-paths"
import { useCurrentUserId } from "@/shared/hooks/use-current-user-id"

export function unreadCountQueryKey(uid: string | undefined) {
  return ["user", uid, "messaging", "unread-count"] as const
}

/** @deprecated Use unreadCountQueryKey(uid) instead */
export const UNREAD_COUNT_QUERY_KEY = ["messaging", "unread-count"]

type UnreadCountResponse = {
  count: number
}

export function useUnreadCount() {
  const uid = useCurrentUserId()

  return useQuery({
    queryKey: unreadCountQueryKey(uid),
    queryFn: () => apiClient<Get<"/api/v1/messaging/unread-count"> & UnreadCountResponse>("/api/v1/messaging/unread-count"),
    staleTime: 30 * 1000,
    refetchInterval: 60 * 1000,
  })
}
