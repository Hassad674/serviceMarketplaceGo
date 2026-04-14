"use client"

import { useQuery } from "@tanstack/react-query"
import { useCurrentUserId } from "@/shared/hooks/use-current-user-id"
import {
  getMyReferrerProfile,
  getPublicReferrerProfile,
  type ReferrerProfile,
} from "../api/referrer-profile-api"

// Query key root matches the invalidation prefix used by the
// organization-shared mutations so a single photo/location/languages
// write fans out to this cache automatically.
export function referrerProfileQueryKey(uid: string | undefined) {
  return ["user", uid, "referrer-profile"] as const
}

export function referrerPublicProfileQueryKey(orgId: string) {
  return ["public", "referrer-profile", orgId] as const
}

// useReferrerProfile reads the authenticated user's referrer profile.
// Auto-creation happens server-side: the first GET materializes the
// row, so the caller never needs to special-case the "doesn't exist
// yet" state.
export function useReferrerProfile() {
  const uid = useCurrentUserId()
  return useQuery<ReferrerProfile>({
    queryKey: referrerProfileQueryKey(uid),
    queryFn: () => getMyReferrerProfile(),
    staleTime: 5 * 60 * 1000,
  })
}

export function usePublicReferrerProfile(orgId: string | undefined) {
  return useQuery<ReferrerProfile>({
    queryKey: orgId ? referrerPublicProfileQueryKey(orgId) : ["noop"],
    queryFn: () => getPublicReferrerProfile(orgId!),
    staleTime: 2 * 60 * 1000,
    enabled: Boolean(orgId),
  })
}
