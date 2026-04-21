"use client"

import { useQuery } from "@tanstack/react-query"
import { getMySubscription } from "../api/subscription-api"
import { subscriptionQueryKey } from "./keys"

/**
 * Reads the authenticated user's current subscription, if any.
 * Returns `null` for free-tier users — the API function squashes
 * the 404 into `null` so the UI never has to catch an error just
 * to distinguish "free" from "premium".
 *
 * The 30-second staleTime matches the navbar badge refresh cadence —
 * Stripe webhooks usually land within a couple of seconds, so a
 * short stale window keeps the UI honest without hammering the API
 * on every navigation.
 */
export function useSubscription() {
  return useQuery({
    queryKey: subscriptionQueryKey.me(),
    queryFn: getMySubscription,
    staleTime: 30_000,
    retry: 1,
  })
}
