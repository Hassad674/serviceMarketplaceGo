"use client"

import { useQuery } from "@tanstack/react-query"
import { useCurrentUserId } from "@/shared/hooks/use-current-user-id"
import { getOrganizationShared } from "../api/organization-shared-api"

// Query key convention: namespaced under the current user so a
// logout/login cycle never serves a stale payload across sessions.
// The split-profile caches use a sibling prefix with the same root so
// invalidation helpers can broadcast a single "user profile changed"
// event.
export function organizationSharedQueryKey(uid: string | undefined) {
  return ["user", uid, "organization-shared"] as const
}

export function useOrganizationShared() {
  const uid = useCurrentUserId()
  return useQuery({
    queryKey: organizationSharedQueryKey(uid),
    queryFn: () => getOrganizationShared(),
    staleTime: 5 * 60 * 1000,
  })
}
