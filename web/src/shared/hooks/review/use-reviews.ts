"use client"

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import {
  fetchReviewsByUser,
  fetchAverageRating,
  fetchCanReview,
  createReview,
  uploadReviewVideo,
  type CreateReviewPayload,
} from "@/shared/lib/review/review-api"
import { useCurrentUserId } from "@/shared/hooks/use-current-user-id"

const REVIEW_KEYS = {
  // Public data — no user scoping needed
  byUser: (userId: string) => ["reviews", "user", userId] as const,
  average: (userId: string) => ["reviews", "average", userId] as const,
  // User-specific — scoped to current user
  canReview: (uid: string | undefined, proposalId: string) =>
    ["user", uid, "reviews", "can-review", proposalId] as const,
}

export function useReviewsByUser(userId: string) {
  return useQuery({
    queryKey: REVIEW_KEYS.byUser(userId),
    queryFn: () => fetchReviewsByUser(userId),
    staleTime: 2 * 60 * 1000,
  })
}

export function useAverageRating(userId: string) {
  return useQuery({
    queryKey: REVIEW_KEYS.average(userId),
    queryFn: () => fetchAverageRating(userId),
    staleTime: 2 * 60 * 1000,
  })
}

export function useCanReview(proposalId: string | undefined) {
  const uid = useCurrentUserId()

  return useQuery({
    queryKey: REVIEW_KEYS.canReview(uid, proposalId ?? ""),
    queryFn: () => fetchCanReview(proposalId!),
    enabled: !!proposalId,
    staleTime: 30 * 1000,
  })
}

export function useCreateReview() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (payload: CreateReviewPayload) => createReview(payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["reviews"] })
    },
  })
}

export function useUploadReviewVideo() {
  return useMutation({
    mutationFn: (file: File) => uploadReviewVideo(file),
  })
}
