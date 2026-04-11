import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import {
  listUsers,
  getUser,
  suspendUser,
  unsuspendUser,
  banUser,
  unbanUser,
  getUserOrganization,
  forceTransferOwnership,
  forceUpdateMemberRole,
  forceRemoveMember,
  forceCancelInvitation,
} from "../api/users-api"
import type {
  SuspendUserPayload,
  BanUserPayload,
  ForceTransferPayload,
  ForceUpdateMemberRolePayload,
} from "../api/users-api"
import type { UserFilters } from "../types"

export function usersQueryKey(filters: UserFilters) {
  return ["admin", "users", filters] as const
}

export function useUsers(filters: UserFilters) {
  return useQuery({
    queryKey: usersQueryKey(filters),
    queryFn: () => listUsers(filters),
    staleTime: 30 * 1000,
  })
}

export function userQueryKey(id: string) {
  return ["admin", "users", id] as const
}

export function useUser(id: string) {
  return useQuery({
    queryKey: userQueryKey(id),
    queryFn: () => getUser(id),
    enabled: !!id,
  })
}

function useInvalidateUser(id: string) {
  const queryClient = useQueryClient()
  return () => {
    queryClient.invalidateQueries({ queryKey: ["admin", "users", id] })
    queryClient.invalidateQueries({ queryKey: ["admin", "users"] })
  }
}

export function useSuspendUser(id: string) {
  const invalidate = useInvalidateUser(id)
  return useMutation({
    mutationFn: (payload: SuspendUserPayload) => suspendUser(id, payload),
    onSuccess: invalidate,
  })
}

export function useUnsuspendUser(id: string) {
  const invalidate = useInvalidateUser(id)
  return useMutation({
    mutationFn: () => unsuspendUser(id),
    onSuccess: invalidate,
  })
}

export function useBanUser(id: string) {
  const invalidate = useInvalidateUser(id)
  return useMutation({
    mutationFn: (payload: BanUserPayload) => banUser(id, payload),
    onSuccess: invalidate,
  })
}

export function useUnbanUser(id: string) {
  const invalidate = useInvalidateUser(id)
  return useMutation({
    mutationFn: () => unbanUser(id),
    onSuccess: invalidate,
  })
}

/* -------------------------------------------------------------------------- */
/* Phase 6 — Team management (admin override)                                 */
/* -------------------------------------------------------------------------- */

export function userOrganizationQueryKey(userID: string) {
  return ["admin", "users", userID, "organization"] as const
}

// Fetches the org detail only when the caller explicitly opts in
// (enabled: true). Used by the team section which conditionally
// renders based on the user's account_type + organization_id, so
// solo providers never trigger a wasted 404 round-trip.
export function useUserOrganization(userID: string, enabled: boolean) {
  return useQuery({
    queryKey: userOrganizationQueryKey(userID),
    queryFn: () => getUserOrganization(userID),
    enabled: !!userID && enabled,
    // The team aggregate is relatively expensive (org + members +
    // invitations in one call) so we keep it fresh for 30s before
    // refetching on window focus.
    staleTime: 30 * 1000,
    // A 404 from the backend is a legitimate outcome (user has no
    // org) so we turn off retries and let the caller render an
    // empty state.
    retry: false,
  })
}

function useInvalidateUserOrganization(userID: string) {
  const queryClient = useQueryClient()
  return () => {
    queryClient.invalidateQueries({ queryKey: userOrganizationQueryKey(userID) })
    // Force-remove / force-transfer also bump session_version on the
    // affected users — their bans/suspension info may have moved.
    queryClient.invalidateQueries({ queryKey: ["admin", "users"] })
  }
}

export function useForceTransferOwnership(userID: string, orgID: string) {
  const invalidate = useInvalidateUserOrganization(userID)
  return useMutation({
    mutationFn: (payload: ForceTransferPayload) => forceTransferOwnership(orgID, payload),
    onSuccess: invalidate,
  })
}

export function useForceUpdateMemberRole(userID: string, orgID: string, targetUserID: string) {
  const invalidate = useInvalidateUserOrganization(userID)
  return useMutation({
    mutationFn: (payload: ForceUpdateMemberRolePayload) =>
      forceUpdateMemberRole(orgID, targetUserID, payload),
    onSuccess: invalidate,
  })
}

export function useForceRemoveMember(userID: string, orgID: string, targetUserID: string) {
  const invalidate = useInvalidateUserOrganization(userID)
  return useMutation({
    mutationFn: () => forceRemoveMember(orgID, targetUserID),
    onSuccess: invalidate,
  })
}

export function useForceCancelInvitation(userID: string, orgID: string, invitationID: string) {
  const invalidate = useInvalidateUserOrganization(userID)
  return useMutation({
    mutationFn: () => forceCancelInvitation(orgID, invitationID),
    onSuccess: invalidate,
  })
}
