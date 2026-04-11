"use client"

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import {
  listMembers,
  listInvitations,
  sendInvitation,
  resendInvitation,
  cancelInvitation,
  updateMember,
  removeMember,
  leaveOrganization,
  initiateTransferOwnership,
  cancelTransferOwnership,
  acceptTransferOwnership,
  declineTransferOwnership,
  validateInvitation,
  acceptInvitation,
} from "../api/team-api"
import type {
  SendInvitationPayload,
  UpdateMemberPayload,
  InitiateTransferPayload,
  AcceptInvitationPayload,
} from "../types"

/* -------------------------------------------------------------------------- */
/* Query keys                                                                 */
/* -------------------------------------------------------------------------- */

// All team-related queries share the ["team", orgID] prefix so one
// invalidation call refreshes every list + every sub-resource at
// once. Mutations that move the session version (promote, remove,
// transfer) additionally invalidate ["session"] from use-user so the
// header/sidebar and the user's permissions catch up.
export function teamMembersKey(orgID: string) {
  return ["team", orgID, "members"] as const
}

export function teamInvitationsKey(orgID: string) {
  return ["team", orgID, "invitations"] as const
}

function useInvalidateTeam(orgID: string) {
  const queryClient = useQueryClient()
  return () => {
    queryClient.invalidateQueries({ queryKey: ["team", orgID] })
  }
}

function useInvalidateTeamAndSession(orgID: string) {
  const queryClient = useQueryClient()
  return () => {
    queryClient.invalidateQueries({ queryKey: ["team", orgID] })
    queryClient.invalidateQueries({ queryKey: ["session"] })
  }
}

/* -------------------------------------------------------------------------- */
/* Reads                                                                      */
/* -------------------------------------------------------------------------- */

export function useTeamMembers(orgID: string | undefined) {
  return useQuery({
    queryKey: teamMembersKey(orgID ?? ""),
    queryFn: () => listMembers(orgID as string),
    enabled: !!orgID,
    staleTime: 30 * 1000,
  })
}

export function useTeamInvitations(orgID: string | undefined) {
  return useQuery({
    queryKey: teamInvitationsKey(orgID ?? ""),
    queryFn: () => listInvitations(orgID as string),
    enabled: !!orgID,
    staleTime: 30 * 1000,
  })
}

/* -------------------------------------------------------------------------- */
/* Invitation mutations                                                       */
/* -------------------------------------------------------------------------- */

export function useSendInvitation(orgID: string) {
  const invalidate = useInvalidateTeam(orgID)
  return useMutation({
    mutationFn: (payload: SendInvitationPayload) => sendInvitation(orgID, payload),
    onSuccess: invalidate,
  })
}

export function useResendInvitation(orgID: string) {
  const invalidate = useInvalidateTeam(orgID)
  return useMutation({
    mutationFn: (invitationID: string) => resendInvitation(orgID, invitationID),
    onSuccess: invalidate,
  })
}

export function useCancelInvitation(orgID: string) {
  const invalidate = useInvalidateTeam(orgID)
  return useMutation({
    mutationFn: (invitationID: string) => cancelInvitation(orgID, invitationID),
    onSuccess: invalidate,
  })
}

/* -------------------------------------------------------------------------- */
/* Membership mutations                                                       */
/* -------------------------------------------------------------------------- */

// Role / title changes bump session_version on the target user so
// their token is invalidated on the next request. That matters for
// the local user only if they happen to be the target — rare but
// possible (Admin editing their own title). Invalidating ["session"]
// is cheap enough that we just always do it.
export function useUpdateMember(orgID: string, userID: string) {
  const invalidate = useInvalidateTeamAndSession(orgID)
  return useMutation({
    mutationFn: (payload: UpdateMemberPayload) => updateMember(orgID, userID, payload),
    onSuccess: invalidate,
  })
}

export function useRemoveMember(orgID: string, userID: string) {
  const invalidate = useInvalidateTeamAndSession(orgID)
  return useMutation({
    mutationFn: () => removeMember(orgID, userID),
    onSuccess: invalidate,
  })
}

// Leave is separate from Remove because the semantics differ — the
// caller is the target, the operator account gets deleted, and
// TanStack Query needs to refetch the session immediately so the
// useOrganization hook flips to null and the sidebar nav item
// disappears without a hard reload.
export function useLeaveOrganization(orgID: string) {
  const invalidate = useInvalidateTeamAndSession(orgID)
  return useMutation({
    mutationFn: () => leaveOrganization(orgID),
    onSuccess: invalidate,
  })
}

/* -------------------------------------------------------------------------- */
/* Ownership transfer                                                         */
/* -------------------------------------------------------------------------- */

export function useInitiateTransfer(orgID: string) {
  const invalidate = useInvalidateTeamAndSession(orgID)
  return useMutation({
    mutationFn: (payload: InitiateTransferPayload) => initiateTransferOwnership(orgID, payload),
    onSuccess: invalidate,
  })
}

export function useCancelTransfer(orgID: string) {
  const invalidate = useInvalidateTeamAndSession(orgID)
  return useMutation({
    mutationFn: () => cancelTransferOwnership(orgID),
    onSuccess: invalidate,
  })
}

export function useAcceptTransfer(orgID: string) {
  const invalidate = useInvalidateTeamAndSession(orgID)
  return useMutation({
    mutationFn: () => acceptTransferOwnership(orgID),
    onSuccess: invalidate,
  })
}

export function useDeclineTransfer(orgID: string) {
  const invalidate = useInvalidateTeamAndSession(orgID)
  return useMutation({
    mutationFn: () => declineTransferOwnership(orgID),
    onSuccess: invalidate,
  })
}

/* -------------------------------------------------------------------------- */
/* Public invitation landing (email link → /invitation/[token])               */
/* -------------------------------------------------------------------------- */

// Powers the public acceptance page. Token validation is a one-shot
// query keyed on the token itself; acceptance is a mutation whose
// success triggers a hard redirect (no TanStack invalidation — the
// whole tree gets re-rendered from scratch with the new cookie).

export function useInvitationPreview(token: string) {
  return useQuery({
    queryKey: ["invitation", "preview", token],
    queryFn: () => validateInvitation(token),
    enabled: !!token,
    retry: false,
    // Preview data is immutable per token — cache it for the whole
    // lifetime of the page so a form re-render doesn't re-fetch.
    staleTime: Infinity,
  })
}

export function useAcceptInvitation() {
  return useMutation({
    mutationFn: (payload: AcceptInvitationPayload) => acceptInvitation(payload),
  })
}
