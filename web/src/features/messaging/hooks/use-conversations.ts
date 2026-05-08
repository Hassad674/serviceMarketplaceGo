"use client"

import { useQuery } from "@tanstack/react-query"
import { listConversations } from "../api/messaging-api"
import { useCurrentUserId } from "@/shared/hooks/use-current-user-id"
// `conversationsQueryKey` is shared with the `proposal` feature (P9).
import { conversationsQueryKey } from "@/shared/lib/query-keys/messaging"
export { conversationsQueryKey }

/** @deprecated Use conversationsQueryKey(uid) instead */
export const CONVERSATIONS_QUERY_KEY = ["messaging", "conversations"]

export function useConversations() {
  const uid = useCurrentUserId()

  return useQuery({
    queryKey: conversationsQueryKey(uid),
    queryFn: () => listConversations(),
    // WS-driven invalidation already keeps the list fresh; polling
    // is a belt-and-braces refresh path. 120 s avoids tripping the
    // global IP rate limit when paired with the rest of the
    // dashboard hooks — see PERF-FIX-W-IDLE-CPU.
    staleTime: 60 * 1000,
    refetchInterval: 120 * 1000,
    refetchIntervalInBackground: false,
  })
}
