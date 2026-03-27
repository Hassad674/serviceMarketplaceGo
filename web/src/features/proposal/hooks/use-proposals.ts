"use client"

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import {
  createProposal,
  getProposal,
  acceptProposal,
  declineProposal,
  modifyProposal,
  simulatePayment,
  requestCompletion,
  completeProposal,
  rejectCompletion,
  listProjects,
} from "../api/proposal-api"
import type { CreateProposalData, ModifyProposalData } from "../api/proposal-api"
import { CONVERSATIONS_QUERY_KEY } from "@/features/messaging/hooks/use-conversations"

export const PROJECTS_QUERY_KEY = ["projects"]
export const PROPOSAL_QUERY_KEY = ["proposal"]

export function useProposal(id: string) {
  return useQuery({
    queryKey: [...PROPOSAL_QUERY_KEY, id],
    queryFn: () => getProposal(id),
    enabled: !!id,
    staleTime: 30 * 1000,
  })
}

export function useCreateProposal() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: CreateProposalData) => createProposal(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: CONVERSATIONS_QUERY_KEY })
    },
  })
}

export function useAcceptProposal() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: string) => acceptProposal(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: CONVERSATIONS_QUERY_KEY })
      queryClient.invalidateQueries({ queryKey: PROJECTS_QUERY_KEY })
      queryClient.invalidateQueries({ queryKey: PROPOSAL_QUERY_KEY })
    },
  })
}

export function useDeclineProposal() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: string) => declineProposal(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: CONVERSATIONS_QUERY_KEY })
      queryClient.invalidateQueries({ queryKey: PROPOSAL_QUERY_KEY })
    },
  })
}

export function useModifyProposal() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: ModifyProposalData }) =>
      modifyProposal(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: CONVERSATIONS_QUERY_KEY })
    },
  })
}

export function useSimulatePayment() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: string) => simulatePayment(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: PROJECTS_QUERY_KEY })
      queryClient.invalidateQueries({ queryKey: CONVERSATIONS_QUERY_KEY })
      queryClient.invalidateQueries({ queryKey: PROPOSAL_QUERY_KEY })
    },
  })
}

export function useRequestCompletion() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: string) => requestCompletion(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: PROJECTS_QUERY_KEY })
      queryClient.invalidateQueries({ queryKey: CONVERSATIONS_QUERY_KEY })
      queryClient.invalidateQueries({ queryKey: PROPOSAL_QUERY_KEY })
    },
  })
}

export function useCompleteProposal() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: string) => completeProposal(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: PROJECTS_QUERY_KEY })
      queryClient.invalidateQueries({ queryKey: CONVERSATIONS_QUERY_KEY })
      queryClient.invalidateQueries({ queryKey: PROPOSAL_QUERY_KEY })
    },
  })
}

export function useRejectCompletion() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: string) => rejectCompletion(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: PROJECTS_QUERY_KEY })
      queryClient.invalidateQueries({ queryKey: CONVERSATIONS_QUERY_KEY })
      queryClient.invalidateQueries({ queryKey: PROPOSAL_QUERY_KEY })
    },
  })
}

export function useProjects(cursor?: string) {
  return useQuery({
    queryKey: [...PROJECTS_QUERY_KEY, cursor],
    queryFn: () => listProjects(cursor),
    staleTime: 30 * 1000,
  })
}
