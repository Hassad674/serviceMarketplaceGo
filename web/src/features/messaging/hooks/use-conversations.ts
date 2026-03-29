"use client"

import { useQuery } from "@tanstack/react-query"
import { listConversations } from "../api/messaging-api"
import { useCurrentUserId } from "@/shared/hooks/use-current-user-id"

export function conversationsQueryKey(uid: string | undefined) {
  return ["user", uid, "messaging", "conversations"] as const
}

/** @deprecated Use conversationsQueryKey(uid) instead */
export const CONVERSATIONS_QUERY_KEY = ["messaging", "conversations"]

export function useConversations() {
  const uid = useCurrentUserId()

  return useQuery({
    queryKey: conversationsQueryKey(uid),
    queryFn: () => listConversations(),
    staleTime: 30 * 1000,
    refetchInterval: 60 * 1000,
  })
}
