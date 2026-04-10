"use client"

import { useQuery } from "@tanstack/react-query"
import { apiClient } from "@/shared/lib/api-client"
import type { AverageRating } from "@/shared/types/review"

type AverageRatingResponse = { data: AverageRating }

/**
 * Local provider-feature hook that fetches the average rating of a user.
 * Duplicates the shape of `useAverageRating` from the review feature to
 * avoid a cross-feature import (features never import each other).
 */
export function useProfileRating(userId: string | undefined) {
  return useQuery({
    queryKey: ["profiles", userId, "average-rating"],
    queryFn: () =>
      apiClient<AverageRatingResponse>(`/api/v1/reviews/average/${userId}`),
    staleTime: 2 * 60 * 1000,
    enabled: !!userId,
    select: (res) => res.data,
  })
}
