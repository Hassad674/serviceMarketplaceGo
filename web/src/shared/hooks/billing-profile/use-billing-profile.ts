"use client"

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import {
  fetchBillingProfile,
  syncBillingProfileFromStripe,
  updateBillingProfile,
  validateBillingProfileVAT,
} from "@/shared/lib/billing-profile/billing-profile-api"
import type {
  BillingProfileSnapshot,
  UpdateBillingProfileInput,
  VIESResult,
} from "@/shared/types/billing-profile"
import { sharedInvoicingQueryKey } from "@/shared/lib/query-keys/invoicing"

/**
 * Shared billing-profile hooks (P9 — the wallet feature renders the
 * completion gate, so the data layer lives in `shared/`).
 *
 * Reads the authenticated organization's billing profile snapshot,
 * including the missing-fields list and the boolean completeness gate.
 *
 * The 30-second `staleTime` matches the cadence at which Stripe KYC
 * webhooks land in the backend — short enough to surface freshly
 * synced data, long enough to avoid a refetch on every page navigation.
 */
export function useBillingProfile() {
  return useQuery<BillingProfileSnapshot>({
    queryKey: sharedInvoicingQueryKey.profile(),
    queryFn: fetchBillingProfile,
    staleTime: 30_000,
    retry: 1,
  })
}

/** PUT mutation. Invalidates the profile cache on success. */
export function useUpdateBillingProfile() {
  const queryClient = useQueryClient()
  return useMutation<BillingProfileSnapshot, Error, UpdateBillingProfileInput>({
    mutationFn: updateBillingProfile,
    onSuccess: (snapshot) => {
      queryClient.setQueryData(sharedInvoicingQueryKey.profile(), snapshot)
      queryClient.invalidateQueries({ queryKey: sharedInvoicingQueryKey.profile() })
    },
  })
}

/** POST sync-from-stripe. Replaces the cached snapshot on success. */
export function useSyncBillingProfile() {
  const queryClient = useQueryClient()
  return useMutation<BillingProfileSnapshot, Error, void>({
    mutationFn: syncBillingProfileFromStripe,
    onSuccess: (snapshot) => {
      queryClient.setQueryData(sharedInvoicingQueryKey.profile(), snapshot)
      queryClient.invalidateQueries({ queryKey: sharedInvoicingQueryKey.profile() })
    },
  })
}

/**
 * POST validate-vat. Refetches the profile on success — the backend
 * stamps `vat_validated_at` on the row, which we want reflected in
 * the form indicator without a manual reload.
 */
export function useValidateVAT() {
  const queryClient = useQueryClient()
  return useMutation<VIESResult, Error, void>({
    mutationFn: validateBillingProfileVAT,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: sharedInvoicingQueryKey.profile() })
    },
  })
}
