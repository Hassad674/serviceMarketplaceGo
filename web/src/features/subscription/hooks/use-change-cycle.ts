"use client"

import { useMutation, useQueryClient } from "@tanstack/react-query"
import { changeCycle } from "../api/subscription-api"
import type { BillingCycle, Subscription } from "../types"
import { subscriptionQueryKey } from "./keys"

/**
 * Switches the current subscription between monthly and annual.
 * Stripe handles the proration server-side; once the mutation
 * returns we invalidate the cached subscription so the new cycle
 * and period-end date flow through to the UI.
 */
export function useChangeCycle() {
  const queryClient = useQueryClient()

  return useMutation<Subscription, Error, BillingCycle>({
    mutationFn: changeCycle,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: subscriptionQueryKey.me() })
      queryClient.invalidateQueries({ queryKey: subscriptionQueryKey.stats() })
    },
  })
}
