"use client"

import { useQuery } from "@tanstack/react-query"
import { apiClient } from "@/shared/lib/api-client"
import type { Get } from "@/shared/lib/api-paths"
import type { AverageRating } from "@/shared/types/review"

type AverageRatingResponse = { data: AverageRating }

// Fetches the average rating of an organization.
// Backend route: GET /api/v1/reviews/average/{orgId}
export function useProfileRating(orgId: string | undefined) {
  return useQuery({
    queryKey: ["profiles", "org", orgId, "average-rating"],
    queryFn: () =>
      apiClient<Get<"/api/v1/reviews/average/{orgId}"> & AverageRatingResponse>(`/api/v1/reviews/average/${orgId}`),
    staleTime: 2 * 60 * 1000,
    enabled: !!orgId,
    select: (res) => res.data,
  })
}
