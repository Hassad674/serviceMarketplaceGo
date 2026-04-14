"use client"

import { useQuery } from "@tanstack/react-query"
import { useCurrentUserId } from "@/shared/hooks/use-current-user-id"
import {
  getReferrerPricing,
  type ReferrerPricing,
} from "../api/referrer-profile-api"

export function referrerPricingQueryKey(uid: string | undefined) {
  return ["user", uid, "referrer-profile", "pricing"] as const
}

export function useReferrerPricing() {
  const uid = useCurrentUserId()
  return useQuery<ReferrerPricing | null>({
    queryKey: referrerPricingQueryKey(uid),
    queryFn: () => getReferrerPricing(),
    staleTime: 5 * 60 * 1000,
  })
}
