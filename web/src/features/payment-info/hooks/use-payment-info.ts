"use client"

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import {
  getPaymentInfo,
  savePaymentInfo,
  getPaymentInfoStatus,
  getRequirements,
} from "../api/payment-info-api"
import type { PaymentInfoFormData } from "../types"
import { useCurrentUserId } from "@/shared/hooks/use-current-user-id"

function paymentInfoKey(uid: string | undefined) {
  return ["user", uid, "payment-info"] as const
}

function paymentInfoStatusKey(uid: string | undefined) {
  return ["user", uid, "payment-info-status"] as const
}

export function usePaymentInfo() {
  const uid = useCurrentUserId()

  return useQuery({
    queryKey: paymentInfoKey(uid),
    queryFn: getPaymentInfo,
    staleTime: 5 * 60 * 1000,
  })
}

export function usePaymentInfoStatus() {
  const uid = useCurrentUserId()

  return useQuery({
    queryKey: paymentInfoStatusKey(uid),
    queryFn: getPaymentInfoStatus,
    staleTime: 5 * 60 * 1000,
  })
}

export function useSavePaymentInfo() {
  const queryClient = useQueryClient()
  const uid = useCurrentUserId()

  return useMutation({
    mutationFn: (input: { data: PaymentInfoFormData; email?: string }) => savePaymentInfo(input.data, input.email),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: paymentInfoKey(uid) })
      queryClient.invalidateQueries({ queryKey: paymentInfoStatusKey(uid) })
    },
  })
}

export function useStripeRequirements() {
  const uid = useCurrentUserId()
  return useQuery({
    queryKey: ["user", uid, "stripe-requirements"],
    queryFn: () => getRequirements(),
    staleTime: 60 * 1000,
    refetchOnWindowFocus: false,
  })
}
