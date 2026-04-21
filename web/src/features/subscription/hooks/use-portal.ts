"use client"

import { useMutation } from "@tanstack/react-query"
import { getPortalURL } from "../api/subscription-api"

/**
 * Exposes the Stripe Customer Portal URL as a mutation (rather
 * than a query) so the caller decides exactly when the request
 * fires — typically on the click of a "Manage payment" or "View
 * invoices" button. Returning the URL lets the button open the
 * portal in a new tab without going through TanStack's cache.
 */
export function usePortalURL() {
  return useMutation<string, Error, void>({
    mutationFn: getPortalURL,
  })
}
