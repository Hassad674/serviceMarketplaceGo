"use client"

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { getWallet, requestPayout, retryFailedTransfer } from "../api/wallet-api"

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
 * Takes the proposal_id from the clicked row and invalidates the wallet
 * cache on success so the badge flips to Transféré without a refresh.
 */
export function useRetryTransfer() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (proposalId: string) => retryFailedTransfer(proposalId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: WALLET_KEY })
    },
  })
}
