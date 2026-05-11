"use client"

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import {
  getWallet,
  getWalletSummary,
  requestPayout,
  retryCommission,
  retryFailedTransfer,
  withdrawWallet,
  type WalletSummary,
  type WithdrawResult,
} from "../api/wallet-api"

const WALLET_KEY = ["wallet"]

/**
 * Query-key factory for the unified wallet surfaces. The legacy
 * `useWallet()` keeps the bare `["wallet"]` key for backwards
 * compatibility; the new unified summary nests under it so a single
 * broad invalidation (`["wallet"]`) refreshes both surfaces after a
 * withdraw.
 */
export const walletKeys = {
  all: WALLET_KEY,
  summary: (cursor?: string) =>
    cursor
      ? (["wallet", "summary", { cursor }] as const)
      : (["wallet", "summary"] as const),
}

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

/**
 * Fetches the unified /wallet/summary envelope. 30 s freshness window
 * to keep dashboard navigation snappy without going stale across
 * tabs. Cursor support is wired through the query key so the consumer
 * can switch pages without confusing the cache.
 */
export function useWalletSummary(cursor?: string) {
  return useQuery<WalletSummary>({
    queryKey: walletKeys.summary(cursor),
    queryFn: () => getWalletSummary(cursor),
    staleTime: 30 * 1000,
  })
}

/**
 * Drains the wallet in a single POST to /wallet/withdraw. The caller
 * handles every branch: 200 success, 207 partial success (errors[]
 * populated on the resolved value), 422 kyc_required (ApiError),
 * 403 billing_profile_incomplete (ApiError). Invalidates the broad
 * ["wallet"] key on settle so both legacy and unified surfaces refresh.
 */
export function useWalletWithdraw() {
  const queryClient = useQueryClient()
  return useMutation<WithdrawResult, Error, number | undefined>({
    mutationFn: (amountCents) => withdrawWallet(amountCents),
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: WALLET_KEY })
    },
  })
}
