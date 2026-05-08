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
    // The websocket pushes `notification_unread_count` events live, so
    // polling is only a fallback. 120 s polling halves the contribution
    // to the global IP rate limit — see PERF-FIX-W-IDLE-CPU.
    staleTime: 60_000,
    refetchInterval: 120_000,
    refetchIntervalInBackground: false,
    select: (data) => data.data.count,
  })
}
