"use client"

import { useMutation, useQueryClient } from "@tanstack/react-query"
import { useCurrentUserId } from "@/shared/hooks/use-current-user-id"
import {
  upsertPricing,
  type Pricing,
  type Profile,
} from "../api/profile-api"
import { profileQueryKey } from "./use-profile"
import { pricingQueryKey } from "./use-pricing"

// Upserts a single pricing row (identified by `kind`). The optimistic
// patch touches two caches:
//   1. The dedicated pricing query (source of truth for the section)
//   2. The profile query (so the public-profile-style strip re-renders)
// On error we roll both back. On success we invalidate — the backend
// may have normalized currency casing or similar.
export function useUpsertPricing() {
  const queryClient = useQueryClient()
  const uid = useCurrentUserId()
  const profileKey = profileQueryKey(uid)
  const pricingKey = pricingQueryKey(uid)

  return useMutation({
    mutationFn: (row: Pricing) => upsertPricing(row),
    onMutate: async (row) => {
      await Promise.all([
        queryClient.cancelQueries({ queryKey: profileKey }),
        queryClient.cancelQueries({ queryKey: pricingKey }),
      ])
      const previousProfile = queryClient.getQueryData<Profile>(profileKey)
      const previousPricing =
        queryClient.getQueryData<Pricing[]>(pricingKey)

      const nextRows = mergePricingRows(previousPricing ?? [], row)

      queryClient.setQueryData<Pricing[]>(pricingKey, nextRows)
      if (previousProfile) {
        queryClient.setQueryData<Profile>(profileKey, {
          ...previousProfile,
          pricing: nextRows,
        })
      }
      return { previousProfile, previousPricing }
    },
    onError: (_error, _row, context) => {
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

function mergePricingRows(existing: Pricing[], incoming: Pricing): Pricing[] {
  const withoutSameKind = existing.filter((row) => row.kind !== incoming.kind)
  return [...withoutSameKind, incoming]
}
