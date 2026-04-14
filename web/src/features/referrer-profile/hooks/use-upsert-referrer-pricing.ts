"use client"

import { useMutation, useQueryClient } from "@tanstack/react-query"
import { useCurrentUserId } from "@/shared/hooks/use-current-user-id"
import {
  upsertReferrerPricing,
  type ReferrerPricing,
  type ReferrerProfile,
  type UpsertReferrerPricingInput,
} from "../api/referrer-profile-api"
import { referrerProfileQueryKey } from "./use-referrer-profile"
import { referrerPricingQueryKey } from "./use-referrer-pricing"

export function useUpsertReferrerPricing() {
  const queryClient = useQueryClient()
  const uid = useCurrentUserId()
  const pricingKey = referrerPricingQueryKey(uid)
  const profileKey = referrerProfileQueryKey(uid)

  return useMutation({
    mutationFn: (input: UpsertReferrerPricingInput) =>
      upsertReferrerPricing(input),
    onMutate: async (input) => {
      await Promise.all([
        queryClient.cancelQueries({ queryKey: pricingKey }),
        queryClient.cancelQueries({ queryKey: profileKey }),
      ])
      const previousPricing =
        queryClient.getQueryData<ReferrerPricing | null>(pricingKey)
      const previousProfile =
        queryClient.getQueryData<ReferrerProfile>(profileKey)

      queryClient.setQueryData<ReferrerPricing | null>(pricingKey, input)
      if (previousProfile) {
        queryClient.setQueryData<ReferrerProfile>(profileKey, {
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
