"use client"

import { useQuery } from "@tanstack/react-query"
import { getUnreadNotificationCount } from "../api/notification-api"
import { useCurrentUserId } from "@/shared/hooks/use-current-user-id"

export function unreadNotifCountKey(uid: string | undefined) {
  return ["user", uid, "notifications", "unread-count"] as const
}

/** @deprecated Use unreadNotifCountKey(uid) instead */
export const UNREAD_NOTIF_COUNT_KEY = ["notifications", "unread-count"]

export function useUnreadNotificationCount() {
  const uid = useCurrentUserId()

  return useQuery({
    queryKey: unreadNotifCountKey(uid),
    queryFn: getUnreadNotificationCount,
    staleTime: 30_000,
    refetchInterval: 60_000,
    select: (data) => data.data.count,
  })
}
