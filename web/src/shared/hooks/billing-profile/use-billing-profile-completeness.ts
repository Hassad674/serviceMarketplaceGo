"use client"

import { useBillingProfile } from "@/shared/hooks/billing-profile/use-billing-profile"
import type { MissingField } from "@/shared/types/billing-profile"

/**
 * Shared completeness gate (P9). Used by wallet (payout pre-flight)
 * and subscription (subscribe pre-flight).
 *
 * Derived gate: should the wallet/subscribe flows let the caller
 * proceed?
 *
 * The truth is owned by the backend's `is_complete` flag — we never
 * recompute completeness on the client. Frontend-only checks would
 * drift from the server contract and let users bypass the gate by
 * stale cache.
 *
 * `isLoading` is true on the very first read so callers can avoid a
 * "false positive" gate during cold start.
 */
export function useBillingProfileCompleteness(): {
  isComplete: boolean
  missingFields: MissingField[]
  isLoading: boolean
  isError: boolean
} {
  const { data, isLoading, isError } = useBillingProfile()
  return {
    isComplete: Boolean(data?.is_complete),
    missingFields: data?.missing_fields ?? [],
    isLoading,
    isError,
  }
}
