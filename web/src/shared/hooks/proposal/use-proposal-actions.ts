"use client"

import { useMutation, useQueryClient } from "@tanstack/react-query"
import {
  acceptProposal,
  declineProposal,
} from "@/shared/lib/proposal/proposal-actions-api"
import { useCurrentUserId } from "@/shared/hooks/use-current-user-id"
import { conversationsQueryKey } from "@/shared/lib/query-keys/messaging"
import {
  projectsQueryKey,
  proposalQueryKey,
} from "@/shared/lib/query-keys/proposal"

/**
 * Shared accept-proposal mutation. Lifted out of the proposal feature
 * (P9) so the messaging feature's `ProposalCard` can wire the action
 * without importing from `@/features/proposal/...`.
 *
 * On success: invalidates conversations + projects + proposal caches,
 * but NOT the messages cache — the WS handler appends the proposal
 * status system message and the proposal-card sync helper updates the
 * card in place.
 */
export function useAcceptProposal() {
  const queryClient = useQueryClient()
  const uid = useCurrentUserId()

  return useMutation({
    mutationFn: (id: string) => acceptProposal(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: conversationsQueryKey(uid) })
      queryClient.invalidateQueries({ queryKey: projectsQueryKey(uid) })
      queryClient.invalidateQueries({ queryKey: proposalQueryKey(uid) })
    },
  })
}

/**
 * Shared decline-proposal mutation. Same rationale as
 * `useAcceptProposal` (cross-feature via messaging).
 */
export function useDeclineProposal() {
  const queryClient = useQueryClient()
  const uid = useCurrentUserId()

  return useMutation({
    mutationFn: (id: string) => declineProposal(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: conversationsQueryKey(uid) })
      queryClient.invalidateQueries({ queryKey: proposalQueryKey(uid) })
    },
  })
}
