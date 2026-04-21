"use client"

import { useMutation, useQueryClient } from "@tanstack/react-query"
import { toggleAutoRenew } from "../api/subscription-api"
import type { Subscription } from "../types"
import { subscriptionQueryKey } from "./keys"

/**
 * Flips `cancel_at_period_end` on the Stripe subscription.
 * Performs an optimistic update so the toggle in the manage modal
 * flips instantly, even on a slow network — if the server rejects
 * the change we roll back to the previous cached subscription.
 */
export function useToggleAutoRenew() {
  const queryClient = useQueryClient()

  return useMutation<
    Subscription,
    Error,
    boolean,
    { previous: Subscription | null | undefined }
  >({
    mutationFn: toggleAutoRenew,
    onMutate: async (autoRenew) => {
      await queryClient.cancelQueries({ queryKey: subscriptionQueryKey.me() })
      const previous = queryClient.getQueryData<Subscription | null>(
        subscriptionQueryKey.me(),
      )
      if (previous) {
        // cancel_at_period_end is the inverse of auto-renew.
        queryClient.setQueryData<Subscription>(subscriptionQueryKey.me(), {
          ...previous,
          cancel_at_period_end: !autoRenew,
        })
      }
      return { previous }
    },
    onError: (_err, _vars, context) => {
      if (context) {
        queryClient.setQueryData(subscriptionQueryKey.me(), context.previous)
      }
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: subscriptionQueryKey.me() })
    },
  })
}
