"use client"

import { useQuery } from "@tanstack/react-query"
import { fetchMyClientProfile } from "../api/client-profile-api"

// useMyClientProfile fetches the authenticated owner's client profile
// via the public `/api/v1/clients/{orgId}` endpoint. Reusing the
// public endpoint here means a single source of truth for the shape
// (company_name, avatar_url, stats) and a single cache key to
// invalidate on writes; `/api/v1/profile` has a different shape
// (provider job title in `title`, client stats nested under `client`)
// and mis-mapping it is error-prone.
//
// The query is disabled until the caller passes a non-empty org id
// so we never hit `/api/v1/clients/` (404) while the session is
// still loading.
export function useMyClientProfile(orgId: string | undefined) {
  return useQuery({
    queryKey: ["client-profile", "me", orgId],
    queryFn: () => fetchMyClientProfile(orgId!),
    enabled: Boolean(orgId),
    staleTime: 5 * 60 * 1000,
  })
}
