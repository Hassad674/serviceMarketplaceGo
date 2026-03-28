"use client"

import { useQuery } from "@tanstack/react-query"
import { getUnreadNotificationCount } from "../api/notification-api"

export const UNREAD_NOTIF_COUNT_KEY = ["notifications", "unread-count"]

export function useUnreadNotificationCount() {
  return useQuery({
    queryKey: UNREAD_NOTIF_COUNT_KEY,
    queryFn: getUnreadNotificationCount,
    staleTime: 30_000,
    refetchInterval: 60_000,
    select: (data) => data.data.count,
  })
}
