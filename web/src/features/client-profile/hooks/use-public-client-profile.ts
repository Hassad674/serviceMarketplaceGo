"use client"

import { useQuery } from "@tanstack/react-query"
import { fetchPublicClientProfile } from "../api/client-profile-api"

// usePublicClientProfile reads the public `/api/v1/clients/{orgId}`
// aggregate. Results are cached for two minutes so navigating back
// and forth between a conversation and a client profile does not
// hammer the backend — the endpoint is fully public and cheap to
// re-hydrate when the window does go stale.
export function usePublicClientProfile(orgId: string | undefined) {
  return useQuery({
    queryKey: ["public-client-profile", orgId],
    queryFn: () => fetchPublicClientProfile(orgId!),
    staleTime: 2 * 60 * 1000,
    enabled: Boolean(orgId),
    retry: false,
  })
}
