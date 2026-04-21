"use client"

import { useMutation } from "@tanstack/react-query"
import { subscribe } from "../api/subscription-api"
import type { SubscribeInput, SubscribeResponse } from "../types"

/**
 * Kicks off the Stripe Checkout flow. On success we redirect the
 * browser to the `checkout_url` returned by the backend — Stripe
 * hosts the PCI-compliant payment form, so we intentionally do
 * not embed it in an iframe.
 */
export function useSubscribe() {
  return useMutation<SubscribeResponse, Error, SubscribeInput>({
    mutationFn: subscribe,
    onSuccess: (data) => {
      // Full-page redirect — the React tree is gone once we leave
      // for Stripe, so there is no cache to invalidate here.
      window.location.href = data.checkout_url
    },
  })
}
