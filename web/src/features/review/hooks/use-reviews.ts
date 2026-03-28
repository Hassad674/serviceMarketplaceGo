"use client"

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import {
  fetchReviewsByUser,
  fetchAverageRating,
  fetchCanReview,
  createReview,
  uploadReviewVideo,
  type CreateReviewPayload,
} from "../api/review-api"

const REVIEW_KEYS = {
  byUser: (userId: string) => ["reviews", "user", userId] as const,
  average: (userId: string) => ["reviews", "average", userId] as const,
  canReview: (proposalId: string) => ["reviews", "can-review", proposalId] as const,
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
  return useQuery({
    queryKey: REVIEW_KEYS.canReview(proposalId ?? ""),
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
