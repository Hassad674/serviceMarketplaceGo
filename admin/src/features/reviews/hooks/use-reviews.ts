import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import {
  listAdminReviews,
  getAdminReview,
  deleteAdminReview,
} from "../api/reviews-api"
import type { ReviewFilters } from "../types"

export function reviewsQueryKey(filters: ReviewFilters) {
  return ["admin", "reviews", filters] as const
}

export function useAdminReviews(filters: ReviewFilters) {
  return useQuery({
    queryKey: reviewsQueryKey(filters),
    queryFn: () => listAdminReviews(filters),
    staleTime: 30 * 1000,
  })
}

export function reviewQueryKey(id: string) {
  return ["admin", "reviews", id] as const
}

export function useAdminReview(id: string) {
  return useQuery({
    queryKey: reviewQueryKey(id),
    queryFn: () => getAdminReview(id),
    enabled: !!id,
  })
}

export function useDeleteReview() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteAdminReview(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin", "reviews"] })
    },
  })
}
