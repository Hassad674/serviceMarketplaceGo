"use client"

import { useQuery, useQueryClient, useMutation } from "@tanstack/react-query"

import {
  getReferral,
  respondToReferral,
} from "@/shared/lib/referral/referral-api"
import type {
  Referral,
  RespondReferralInput,
} from "@/shared/types/referral"

/**
 * Shared subset of the referral query keys. Mirrors the
 * `referralKeys.detail(id)` shape from
 * `features/referral/hooks/use-referrals` so cache writes stay in sync
 * across both surfaces. The full dashboard tree
 * (`referralKeys.myList(filter)` etc.) stays in the feature.
 */
export const sharedReferralKeys = {
  all: ["referrals"] as const,
  detail: (id: string) => ["referrals", "detail", id] as const,
  negotiations: (id: string) => ["referrals", "negotiations", id] as const,
}

/**
 * Shared `useReferral` hook (P9). Used by `ReferralSystemMessage`,
 * which the messaging feature embeds in conversation timelines. Polls
 * every 5 s while the row is in a pending state so the inline card
 * reflects the other party's response without a manual refresh.
 */
export function useReferral(id: string | undefined) {
  return useQuery<Referral>({
    queryKey: id ? sharedReferralKeys.detail(id) : ["referrals", "detail", "noop"],
    queryFn: () => getReferral(id!),
    enabled: Boolean(id),
    staleTime: 5 * 1000,
    refetchInterval: (query) => {
      const status = query.state.data?.status
      if (!status) return false
      return status.startsWith("pending_") ? 5000 : false
    },
  })
}

/**
 * Shared `useRespondToReferral` mutation (P9). Used by `ReferralActions`.
 * Invalidates the detail query and the dashboard subtree so all surfaces
 * stay in sync after a state transition.
 */
export function useRespondToReferral(id: string | undefined) {
  const queryClient = useQueryClient()
  return useMutation<Referral, Error, RespondReferralInput>({
    mutationFn: (input) => {
      if (!id) throw new Error("referral id is required")
      return respondToReferral(id, input)
    },
    onSuccess: (data) => {
      if (id) {
        queryClient.setQueryData(sharedReferralKeys.detail(id), data)
        queryClient.invalidateQueries({ queryKey: sharedReferralKeys.negotiations(id) })
      }
      queryClient.invalidateQueries({ queryKey: sharedReferralKeys.all })
    },
  })
}
