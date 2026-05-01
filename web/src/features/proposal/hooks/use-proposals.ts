"use client"

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import {
  createProposal,
  getProposal,
  acceptProposal,
  declineProposal,
  modifyProposal,
  initiatePayment,
  submitMilestone,
  approveMilestone,
  rejectMilestone,
  listProjects,
} from "../api/proposal-api"
import type { CreateProposalData, ModifyProposalData } from "../api/proposal-api"
import { conversationsQueryKey } from "@/features/messaging/hooks/use-conversations"
import { messagesQueryKey } from "@/features/messaging/hooks/use-messages"
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

// Phase 11: per-milestone action hooks. Each mutation carries the
// explicit milestone id so the backend can optimistic-lock against a
// concrete row, instead of implicitly mutating the "current active
// milestone" (which drifts when two tabs race). All three invalidate
// the same set of queries as the legacy per-proposal hooks they
// replaced so the stepper, project list, and conversation list all
// refresh after a state transition.

type MilestoneActionInput = {
  proposalID: string
  milestoneID: string
}

function invalidateProposalCaches(queryClient: ReturnType<typeof useQueryClient>, uid: string | undefined) {
  queryClient.invalidateQueries({ queryKey: projectsQueryKey(uid) })
  queryClient.invalidateQueries({ queryKey: conversationsQueryKey(uid) })
  queryClient.invalidateQueries({ queryKey: proposalQueryKey(uid) })
}

export function useSubmitMilestone() {
  const queryClient = useQueryClient()
  const uid = useCurrentUserId()

  return useMutation({
    mutationFn: ({ proposalID, milestoneID }: MilestoneActionInput) =>
      submitMilestone(proposalID, milestoneID),
    onSuccess: () => invalidateProposalCaches(queryClient, uid),
  })
}

export function useApproveMilestone() {
  const queryClient = useQueryClient()
  const uid = useCurrentUserId()

  return useMutation({
    mutationFn: ({ proposalID, milestoneID }: MilestoneActionInput) =>
      approveMilestone(proposalID, milestoneID),
    onSuccess: () => invalidateProposalCaches(queryClient, uid),
  })
}

export function useRejectMilestone() {
  const queryClient = useQueryClient()
  const uid = useCurrentUserId()

  return useMutation({
    mutationFn: ({ proposalID, milestoneID }: MilestoneActionInput) =>
      rejectMilestone(proposalID, milestoneID),
    onSuccess: () => invalidateProposalCaches(queryClient, uid),
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
