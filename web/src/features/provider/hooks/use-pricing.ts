"use client"

import { useQuery } from "@tanstack/react-query"
import { useCurrentUserId } from "@/shared/hooks/use-current-user-id"
import { getPricing, type Pricing } from "../api/profile-api"

export function pricingQueryKey(uid: string | undefined) {
  return ["user", uid, "profile", "pricing"] as const
}

// Dedicated query for the pricing rows. They are also embedded in the
// ProfileResponse, but the dedicated endpoint is the single source of
// truth — the `useProfile()` cache is treated as a hint for the initial
// render only. We keep a 5 min stale time to match the profile query.
export function usePricing() {
  const uid = useCurrentUserId()

  return useQuery({
    queryKey: pricingQueryKey(uid),
    queryFn: () => getPricing(),
    staleTime: 5 * 60 * 1000,
  })
}

export type { Pricing }
