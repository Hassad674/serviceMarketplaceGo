"use client"

import { useQuery } from "@tanstack/react-query"
import { useCurrentUserId } from "@/shared/hooks/use-current-user-id"
import {
  getMyFreelanceProfile,
  getPublicFreelanceProfile,
  type FreelanceProfile,
} from "../api/freelance-profile-api"

// Query key convention matches the organization-shared feature so a
// single "user profile changed" invalidation can fan out to every
// persona cache. The root is ["user", uid, "freelance-profile"],
// matching the prefix referenced by invalidateSharedDependents.
export function freelanceProfileQueryKey(uid: string | undefined) {
  return ["user", uid, "freelance-profile"] as const
}

export function freelancePublicProfileQueryKey(orgId: string) {
  return ["public", "freelance-profile", orgId] as const
}

// useFreelanceProfile reads the authenticated user's freelance
// profile. Auto-creation happens on the backend side: the first GET
// after migration materializes the row from the legacy provider
// profile, so the caller never needs a "does it exist?" check.
export function useFreelanceProfile() {
  const uid = useCurrentUserId()
  return useQuery<FreelanceProfile>({
    queryKey: freelanceProfileQueryKey(uid),
    queryFn: () => getMyFreelanceProfile(),
    staleTime: 5 * 60 * 1000,
  })
}

// usePublicFreelanceProfile reads any organization's freelance
// profile. Shorter stale time because the public viewer is more
// latency-sensitive — if the org just updated their title, visitors
// should see it quickly.
export function usePublicFreelanceProfile(orgId: string | undefined) {
  return useQuery<FreelanceProfile>({
    queryKey: orgId ? freelancePublicProfileQueryKey(orgId) : ["noop"],
    queryFn: () => getPublicFreelanceProfile(orgId!),
    staleTime: 2 * 60 * 1000,
    enabled: Boolean(orgId),
  })
}
