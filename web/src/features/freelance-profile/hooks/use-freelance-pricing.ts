"use client"

import { useQuery } from "@tanstack/react-query"
import { useCurrentUserId } from "@/shared/hooks/use-current-user-id"
import {
  getFreelancePricing,
  type FreelancePricing,
} from "../api/freelance-profile-api"

export function freelancePricingQueryKey(uid: string | undefined) {
  return ["user", uid, "freelance-profile", "pricing"] as const
}

// useFreelancePricing reads the dedicated pricing row. The freelance
// profile response already embeds this row, but the dedicated
// endpoint is the source of truth — the profile cache is a hint only.
// Matches the agency pattern so rendering/saving UI stays symmetrical
// across personas.
export function useFreelancePricing() {
  const uid = useCurrentUserId()
  return useQuery<FreelancePricing | null>({
    queryKey: freelancePricingQueryKey(uid),
    queryFn: () => getFreelancePricing(),
    staleTime: 5 * 60 * 1000,
  })
}
