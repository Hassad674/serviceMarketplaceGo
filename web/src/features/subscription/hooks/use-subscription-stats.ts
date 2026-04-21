"use client"

import { useQuery } from "@tanstack/react-query"
import { getStats } from "../api/subscription-api"
import { subscriptionQueryKey } from "./keys"

/**
 * Reads the "savings since you became Premium" stats. Returns
 * `null` for free-tier users (the API translates the 404 into
 * `null`). Consumers render the savings line only when the stats
 * object is present.
 */
export function useSubscriptionStats() {
  return useQuery({
    queryKey: subscriptionQueryKey.stats(),
    queryFn: getStats,
    staleTime: 60_000,
  })
}
