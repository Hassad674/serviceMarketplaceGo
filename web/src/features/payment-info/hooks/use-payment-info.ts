"use client"

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import {
  getPaymentInfo,
  savePaymentInfo,
  getPaymentInfoStatus,
} from "../api/payment-info-api"
import type { PaymentInfoFormData } from "../types"

const PAYMENT_INFO_KEY = ["payment-info"]
const PAYMENT_INFO_STATUS_KEY = ["payment-info-status"]

export function usePaymentInfo() {
  return useQuery({
    queryKey: PAYMENT_INFO_KEY,
    queryFn: getPaymentInfo,
    staleTime: 5 * 60 * 1000,
  })
}

export function usePaymentInfoStatus() {
  return useQuery({
    queryKey: PAYMENT_INFO_STATUS_KEY,
    queryFn: getPaymentInfoStatus,
    staleTime: 5 * 60 * 1000,
  })
}

export function useSavePaymentInfo() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: PaymentInfoFormData) => savePaymentInfo(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: PAYMENT_INFO_KEY })
      queryClient.invalidateQueries({ queryKey: PAYMENT_INFO_STATUS_KEY })
    },
  })
}
