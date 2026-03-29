"use client"

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import {
  createProposal,
  getProposal,
  acceptProposal,
  declineProposal,
  modifyProposal,
  initiatePayment,
  requestCompletion,
  completeProposal,
  rejectCompletion,
  listProjects,
} from "../api/proposal-api"
import type { CreateProposalData, ModifyProposalData } from "../api/proposal-api"
import { conversationsQueryKey } from "@/features/messaging/hooks/use-conversations"
import { messagesQueryKey, MESSAGES_KEY_BASE } from "@/features/messaging/hooks/use-messages"
import { useCurrentUserId } from "@/shared/hooks/use-current-user-id"

export function projectsQueryKey(uid: string | undefined) {
  return ["user", uid, "projects"] as const
}

export function proposalQueryKey(uid: string | undefined) {
  return ["user", uid, "proposal"] as const
}

/** @deprecated Use projectsQueryKey(uid) instead */
export const PROJECTS_QUERY_KEY = ["projects"]
/** @deprecated Use proposalQueryKey(uid) instead */
export const PROPOSAL_QUERY_KEY = ["proposal"]

export function useProposal(id: string) {
  const uid = useCurrentUserId()

  return useQuery({
    queryKey: [...proposalQueryKey(uid), id],
    queryFn: () => getProposal(id),
    enabled: !!id,
    staleTime: 30 * 1000,
  })
}

export function useCreateProposal() {
  const queryClient = useQueryClient()
  const uid = useCurrentUserId()

  return useMutation({
    mutationFn: (data: CreateProposalData) => createProposal(data),
    onSuccess: (_result, variables) => {
      queryClient.invalidateQueries({ queryKey: conversationsQueryKey(uid) })
      // Invalidate messages for the target conversation so the sender
      // sees the proposal_sent message immediately after navigating back.
      queryClient.invalidateQueries({
        queryKey: messagesQueryKey(uid, variables.conversation_id),
      })
    },
  })
}

export function useAcceptProposal() {
  const queryClient = useQueryClient()
  const uid = useCurrentUserId()

  return useMutation({
    mutationFn: (id: string) => acceptProposal(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: conversationsQueryKey(uid) })
      queryClient.invalidateQueries({ queryKey: projectsQueryKey(uid) })
      queryClient.invalidateQueries({ queryKey: proposalQueryKey(uid) })
      // Do NOT invalidate messages here. The WS handler adds the
      // system message and syncProposalStatusInCache updates proposal cards.
    },
  })
}

export function useDeclineProposal() {
  const queryClient = useQueryClient()
  const uid = useCurrentUserId()

  return useMutation({
    mutationFn: (id: string) => declineProposal(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: conversationsQueryKey(uid) })
      queryClient.invalidateQueries({ queryKey: proposalQueryKey(uid) })
      // Do NOT invalidate messages -- same reason as useAcceptProposal.
    },
  })
}

export function useModifyProposal() {
  const queryClient = useQueryClient()
  const uid = useCurrentUserId()

  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: ModifyProposalData }) =>
      modifyProposal(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: conversationsQueryKey(uid) })
    },
  })
}

export function useInitiatePayment() {
  const queryClient = useQueryClient()
  const uid = useCurrentUserId()

  return useMutation({
    mutationFn: (id: string) => initiatePayment(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: projectsQueryKey(uid) })
      queryClient.invalidateQueries({ queryKey: conversationsQueryKey(uid) })
      queryClient.invalidateQueries({ queryKey: proposalQueryKey(uid) })
    },
  })
}

export function useRequestCompletion() {
  const queryClient = useQueryClient()
  const uid = useCurrentUserId()

  return useMutation({
    mutationFn: (id: string) => requestCompletion(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: projectsQueryKey(uid) })
      queryClient.invalidateQueries({ queryKey: conversationsQueryKey(uid) })
      queryClient.invalidateQueries({ queryKey: proposalQueryKey(uid) })
    },
  })
}

export function useCompleteProposal() {
  const queryClient = useQueryClient()
  const uid = useCurrentUserId()

  return useMutation({
    mutationFn: (id: string) => completeProposal(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: projectsQueryKey(uid) })
      queryClient.invalidateQueries({ queryKey: conversationsQueryKey(uid) })
      queryClient.invalidateQueries({ queryKey: proposalQueryKey(uid) })
    },
  })
}

export function useRejectCompletion() {
  const queryClient = useQueryClient()
  const uid = useCurrentUserId()

  return useMutation({
    mutationFn: (id: string) => rejectCompletion(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: projectsQueryKey(uid) })
      queryClient.invalidateQueries({ queryKey: conversationsQueryKey(uid) })
      queryClient.invalidateQueries({ queryKey: proposalQueryKey(uid) })
    },
  })
}

export function useProjects(cursor?: string) {
  const uid = useCurrentUserId()

  return useQuery({
    queryKey: [...projectsQueryKey(uid), cursor],
    queryFn: () => listProjects(cursor),
    staleTime: 30 * 1000,
  })
}
