"use client"

import { useQuery } from "@tanstack/react-query"
import { apiClient } from "@/shared/lib/api-client"
import type { Get } from "@/shared/lib/api-paths"
import type { AverageRating } from "@/shared/types/review"

type AverageRatingResponse = { data: AverageRating }

// useProfileRating reads the aggregate rating for an organization.
// Shared across every profile persona (agency, freelance, referrer)
// because the backend keys the endpoint on organization id and
// returns the same shape regardless of persona.
export function useProfileRating(orgId: string | undefined) {
  return useQuery({
    queryKey: ["profiles", "org", orgId, "average-rating"],
    queryFn: () =>
      apiClient<Get<"/api/v1/reviews/average/{orgId}"> & AverageRatingResponse>(`/api/v1/reviews/average/${orgId}`),
    staleTime: 2 * 60 * 1000,
    enabled: Boolean(orgId),
    select: (res) => res.data,
  })
}
