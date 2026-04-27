"use client"

import { useMutation } from "@tanstack/react-query"
import { subscribe } from "../api/subscription-api"
import type { SubscribeInput, SubscribeResponse } from "../types"

/**
 * Creates a Stripe Embedded Checkout session and returns the
 * `client_secret` the caller mounts via @stripe/react-stripe-js.
 *
 * Unlike the legacy hosted-URL flow, this hook DOES NOT navigate
 * the browser. The caller controls when to render the embedded
 * payment form (typically after the billing-profile step inside
 * the upgrade modal). The embedded form itself handles the
 * redirect to ReturnURL once the user pays.
 */
export function useSubscribe() {
  return useMutation<SubscribeResponse, Error, SubscribeInput>({
    mutationFn: subscribe,
  })
}
