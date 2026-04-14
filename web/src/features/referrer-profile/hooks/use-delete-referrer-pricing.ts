"use client"

import { useMutation, useQueryClient } from "@tanstack/react-query"
import { useCurrentUserId } from "@/shared/hooks/use-current-user-id"
import {
  deleteReferrerPricing,
  type ReferrerPricing,
  type ReferrerProfile,
} from "../api/referrer-profile-api"
import { referrerProfileQueryKey } from "./use-referrer-profile"
import { referrerPricingQueryKey } from "./use-referrer-pricing"

export function useDeleteReferrerPricing() {
  const queryClient = useQueryClient()
  const uid = useCurrentUserId()
  const pricingKey = referrerPricingQueryKey(uid)
  const profileKey = referrerProfileQueryKey(uid)

  return useMutation({
    mutationFn: () => deleteReferrerPricing(),
    onMutate: async () => {
      await Promise.all([
        queryClient.cancelQueries({ queryKey: pricingKey }),
        queryClient.cancelQueries({ queryKey: profileKey }),
      ])
      const previousPricing =
        queryClient.getQueryData<ReferrerPricing | null>(pricingKey)
      const previousProfile =
        queryClient.getQueryData<ReferrerProfile>(profileKey)

      queryClient.setQueryData<ReferrerPricing | null>(pricingKey, null)
      if (previousProfile) {
        queryClient.setQueryData<ReferrerProfile>(profileKey, {
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
