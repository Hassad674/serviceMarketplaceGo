"use client"

import { useQuery } from "@tanstack/react-query"
import { fetchMyClientProfile } from "../api/client-profile-api"

// useMyClientProfile fetches the authenticated user's own client
// profile slice of /api/v1/profile. We intentionally do NOT share the
// provider feature's `useProfile()` hook here — the client-profile
// feature must compile and render even when the provider feature is
// removed (feature isolation rule from CLAUDE.md). Both hooks hit the
// same endpoint but keep their own query keys so each feature owns
// its cache invalidation contract.
export function useMyClientProfile() {
  return useQuery({
    queryKey: ["client-profile", "me"],
    queryFn: () => fetchMyClientProfile(),
    staleTime: 5 * 60 * 1000,
  })
}
