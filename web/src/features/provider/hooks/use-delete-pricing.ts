"use client"

import { useMutation, useQueryClient } from "@tanstack/react-query"
import { useCurrentUserId } from "@/shared/hooks/use-current-user-id"
import {
  deletePricing,
  type Pricing,
  type PricingKind,
  type Profile,
} from "../api/profile-api"
import { profileQueryKey } from "./use-profile"
import { pricingQueryKey } from "./use-pricing"

// Removes a single pricing row by kind. Optimistic removal from both
// the dedicated pricing cache and the embedded profile copy.
export function useDeletePricing() {
  const queryClient = useQueryClient()
  const uid = useCurrentUserId()
  const profileKey = profileQueryKey(uid)
  const pricingKey = pricingQueryKey(uid)

  return useMutation({
    mutationFn: (kind: PricingKind) => deletePricing(kind),
    onMutate: async (kind) => {
      await Promise.all([
        queryClient.cancelQueries({ queryKey: profileKey }),
        queryClient.cancelQueries({ queryKey: pricingKey }),
      ])
      const previousProfile = queryClient.getQueryData<Profile>(profileKey)
      const previousPricing =
        queryClient.getQueryData<Pricing[]>(pricingKey)

      const nextRows = (previousPricing ?? []).filter((row) => row.kind !== kind)
      queryClient.setQueryData<Pricing[]>(pricingKey, nextRows)
      if (previousProfile) {
        queryClient.setQueryData<Profile>(profileKey, {
          ...previousProfile,
          pricing: nextRows,
        })
      }
      return { previousProfile, previousPricing }
    },
    onError: (_error, _kind, context) => {
      if (context?.previousProfile) {
        queryClient.setQueryData<Profile>(profileKey, context.previousProfile)
      }
      if (context?.previousPricing !== undefined) {
        queryClient.setQueryData<Pricing[]>(pricingKey, context.previousPricing)
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: pricingKey })
      queryClient.invalidateQueries({ queryKey: profileKey })
    },
  })
}
