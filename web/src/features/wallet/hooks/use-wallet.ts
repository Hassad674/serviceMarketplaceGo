"use client"

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import {
  getWallet,
  requestPayout,
  retryCommission,
  retryFailedTransfer,
} from "../api/wallet-api"

const WALLET_KEY = ["wallet"]

export function useWallet() {
  return useQuery({
    queryKey: WALLET_KEY,
    queryFn: getWallet,
  })
}

export function useRequestPayout() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: requestPayout,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: WALLET_KEY })
    },
  })
}

/**
 * Mutation for the per-record "Retry" action shown on failed transfers.
 * Takes the record.id from the clicked row (NOT proposal_id — a proposal
 * can have multiple records, one per milestone, so proposal_id is
 * ambiguous for retry targeting) and invalidates the wallet cache on
 * success so the badge flips to Transféré without a refresh.
 */
export function useRetryTransfer() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (recordId: string) => retryFailedTransfer(recordId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: WALLET_KEY })
    },
  })
}

/**
 * Mutation for the apporteur-side "Retirer" button on a commission
 * row stuck in pending_kyc or failed (D1+D2). Takes the commission id
 * (NOT the milestone id — the commission row is the unambiguous
 * target). The caller is responsible for handling the 422
 * kyc_required branch by inspecting the rejected error and opening
 * the onboarding modal. On 200 (paid) we invalidate the wallet
 * query so the row flips to "Reçue" without a refresh.
 */
export function useRetryCommission() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (commissionId: string) => retryCommission(commissionId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: WALLET_KEY })
    },
  })
}
