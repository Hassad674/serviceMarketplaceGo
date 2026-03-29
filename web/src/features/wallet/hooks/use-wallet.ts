"use client"

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { getWallet, requestPayout } from "../api/wallet-api"

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
