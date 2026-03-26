"use client"

import { useQuery } from "@tanstack/react-query"
import { apiClient } from "@/shared/lib/api-client"

export const UNREAD_COUNT_QUERY_KEY = ["messaging", "unread-count"]

type UnreadCountResponse = {
  count: number
}

export function useUnreadCount() {
  return useQuery({
    queryKey: UNREAD_COUNT_QUERY_KEY,
    queryFn: () => apiClient<UnreadCountResponse>("/api/v1/messaging/unread-count"),
    staleTime: 30 * 1000,
    refetchInterval: 60 * 1000,
  })
}
