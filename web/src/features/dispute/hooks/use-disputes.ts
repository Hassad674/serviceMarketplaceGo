"use client"

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import {
  openDispute,
  getDispute,
  counterPropose,
  respondToCounter,
  cancelDispute,
  respondToCancellation,
} from "../api/dispute-api"
import type { OpenDisputeData, CounterProposeData } from "../api/dispute-api"

const DISPUTE_KEY = ["dispute"]
const PROJECTS_KEY = ["projects"]
const PROPOSALS_KEY = ["proposals"]

export function useDispute(id: string | undefined) {
  return useQuery({
    queryKey: [...DISPUTE_KEY, id],
    queryFn: () => getDispute(id!),
    enabled: !!id,
    staleTime: 30_000,
  })
}

export function useOpenDispute() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: OpenDisputeData) => openDispute(data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: PROJECTS_KEY })
      qc.invalidateQueries({ queryKey: PROPOSALS_KEY })
    },
  })
}

export function useCounterPropose(disputeId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: CounterProposeData) => counterPropose(disputeId, data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: [...DISPUTE_KEY, disputeId] })
      qc.invalidateQueries({ queryKey: PROJECTS_KEY })
    },
  })
}

export function useRespondToCounter(disputeId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ cpId, accept }: { cpId: string; accept: boolean }) =>
      respondToCounter(disputeId, cpId, accept),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: [...DISPUTE_KEY, disputeId] })
      qc.invalidateQueries({ queryKey: PROJECTS_KEY })
      qc.invalidateQueries({ queryKey: PROPOSALS_KEY })
    },
  })
}

export function useCancelDispute() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => cancelDispute(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: DISPUTE_KEY })
      qc.invalidateQueries({ queryKey: PROJECTS_KEY })
      qc.invalidateQueries({ queryKey: PROPOSALS_KEY })
    },
  })
}

export function useRespondToCancellation(disputeId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (accept: boolean) => respondToCancellation(disputeId, accept),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: [...DISPUTE_KEY, disputeId] })
      qc.invalidateQueries({ queryKey: DISPUTE_KEY })
      qc.invalidateQueries({ queryKey: PROJECTS_KEY })
      qc.invalidateQueries({ queryKey: PROPOSALS_KEY })
    },
  })
}
