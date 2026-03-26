"use client"

import { useQuery } from "@tanstack/react-query"
import { listConversations } from "../api/messaging-api"

export const CONVERSATIONS_QUERY_KEY = ["messaging", "conversations"]

export function useConversations() {
  return useQuery({
    queryKey: CONVERSATIONS_QUERY_KEY,
    queryFn: () => listConversations(),
    staleTime: 30 * 1000,
    refetchInterval: 60 * 1000,
  })
}
