"use client"

import { useMutation, useQueryClient } from "@tanstack/react-query"
import { useCurrentUserId } from "@/shared/hooks/use-current-user-id"
import {
  upsertFreelancePricing,
  type FreelancePricing,
  type UpsertFreelancePricingInput,
} from "../api/freelance-profile-api"
import {
  freelanceProfileQueryKey,
} from "./use-freelance-profile"
import { freelancePricingQueryKey } from "./use-freelance-pricing"
import type { FreelanceProfile } from "../api/freelance-profile-api"

// Optimistic upsert: patches both the dedicated pricing cache and the
// embedded pricing on the profile cache so the identity strip and the
// pricing section update in the same frame. Rolls back both on error.
export function useUpsertFreelancePricing() {
  const queryClient = useQueryClient()
  const uid = useCurrentUserId()
  const pricingKey = freelancePricingQueryKey(uid)
  const profileKey = freelanceProfileQueryKey(uid)

  return useMutation({
    mutationFn: (input: UpsertFreelancePricingInput) =>
      upsertFreelancePricing(input),
    onMutate: async (input) => {
      await Promise.all([
        queryClient.cancelQueries({ queryKey: pricingKey }),
        queryClient.cancelQueries({ queryKey: profileKey }),
      ])
      const previousPricing =
        queryClient.getQueryData<FreelancePricing | null>(pricingKey)
      const previousProfile =
        queryClient.getQueryData<FreelanceProfile>(profileKey)

      queryClient.setQueryData<FreelancePricing | null>(pricingKey, input)
      if (previousProfile) {
        queryClient.setQueryData<FreelanceProfile>(profileKey, {
          ...previousProfile,
          pricing: input,
        })
      }
      return { previousPricing, previousProfile }
    },
    onError: (_err, _input, ctx) => {
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
