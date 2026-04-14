"use client"

import { useMutation, useQueryClient } from "@tanstack/react-query"
import { useCurrentUserId } from "@/shared/hooks/use-current-user-id"
import {
  updateOrganizationLocation,
  type OrganizationSharedProfile,
  type UpdateOrganizationLocationInput,
} from "../api/organization-shared-api"
import { organizationSharedQueryKey } from "./use-organization-shared"

// Writes the location block to the org row, then invalidates every
// cached query that surfaces shared fields — the split-profile reads
// decorate their response with the same block via JOIN, so their
// cached payloads must be refreshed in lockstep. Matching is done by
// key prefix so each persona feature only has to add its own root to
// the SHARED_PROFILE_DEPENDENT_PREFIXES constant below.
const SHARED_PROFILE_DEPENDENT_PREFIXES = [
  "freelance-profile",
  "referrer-profile",
] as const

export function useUpdateOrganizationLocation() {
  const queryClient = useQueryClient()
  const uid = useCurrentUserId()
  const sharedKey = organizationSharedQueryKey(uid)

  return useMutation({
    mutationFn: (input: UpdateOrganizationLocationInput) =>
      updateOrganizationLocation(input),
    onSuccess: (next) => {
      queryClient.setQueryData<OrganizationSharedProfile>(sharedKey, next)
      invalidateSharedDependents(queryClient, uid)
    },
  })
}

// Helper exported for the other shared mutations so they all fan out
// the same way — single rule change, all personas refresh.
export function invalidateSharedDependents(
  queryClient: ReturnType<typeof useQueryClient>,
  uid: string | undefined,
) {
  for (const prefix of SHARED_PROFILE_DEPENDENT_PREFIXES) {
    queryClient.invalidateQueries({
      queryKey: ["user", uid, prefix],
    })
  }
}
