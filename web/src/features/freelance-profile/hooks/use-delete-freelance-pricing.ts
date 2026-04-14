"use client"

import { useMutation, useQueryClient } from "@tanstack/react-query"
import { useCurrentUserId } from "@/shared/hooks/use-current-user-id"
import {
  deleteFreelancePricing,
  type FreelancePricing,
  type FreelanceProfile,
} from "../api/freelance-profile-api"
import { freelanceProfileQueryKey } from "./use-freelance-profile"
import { freelancePricingQueryKey } from "./use-freelance-pricing"

// Optimistic delete: nulls the pricing row in both caches, rolls back
// on error, invalidates on success to pick up any backend side-effects.
export function useDeleteFreelancePricing() {
  const queryClient = useQueryClient()
  const uid = useCurrentUserId()
  const pricingKey = freelancePricingQueryKey(uid)
  const profileKey = freelanceProfileQueryKey(uid)

  return useMutation({
    mutationFn: () => deleteFreelancePricing(),
    onMutate: async () => {
      await Promise.all([
        queryClient.cancelQueries({ queryKey: pricingKey }),
        queryClient.cancelQueries({ queryKey: profileKey }),
      ])
      const previousPricing =
        queryClient.getQueryData<FreelancePricing | null>(pricingKey)
      const previousProfile =
        queryClient.getQueryData<FreelanceProfile>(profileKey)

      queryClient.setQueryData<FreelancePricing | null>(pricingKey, null)
      if (previousProfile) {
        queryClient.setQueryData<FreelanceProfile>(profileKey, {
          ...previousProfile,
          pricing: null,
        })
      }
      return { previousPricing, previousProfile }
    },
    onError: (_err, _vars, ctx) => {
      if (ctx?.previousPricing !== undefined) {
        queryClient.setQueryData(pricingKey, ctx.previousPricing)
      }
      if (ctx?.previousProfile) {
        queryClient.setQueryData(profileKey, ctx.previousProfile)
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: pricingKey })
      queryClient.invalidateQueries({ queryKey: profileKey })
    },
  })
}
