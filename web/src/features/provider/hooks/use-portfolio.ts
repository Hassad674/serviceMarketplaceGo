"use client"

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { useCurrentUserId } from "@/shared/hooks/use-current-user-id"
import {
  fetchPortfolioByUser,
  createPortfolioItem,
  updatePortfolioItem,
  deletePortfolioItem,
  reorderPortfolio,
  uploadPortfolioImage,
  uploadPortfolioVideo,
} from "../api/portfolio-api"
type MediaPayload = {
  media_url: string
  media_type: string
  thumbnail_url?: string
  position: number
}

function myPortfolioKey(uid: string | undefined) {
  return ["user", uid, "my-portfolio"] as const
}

export function useMyPortfolio() {
  const uid = useCurrentUserId()

  return useQuery({
    queryKey: myPortfolioKey(uid),
    queryFn: async () => {
      if (!uid) return { data: [], next_cursor: "", has_more: false }
      return fetchPortfolioByUser(uid)
    },
    staleTime: 2 * 60 * 1000,
    enabled: !!uid,
  })
}

export function usePortfolioByUser(userId: string) {
  return useQuery({
    queryKey: ["portfolio", "user", userId],
    queryFn: () => fetchPortfolioByUser(userId),
    staleTime: 2 * 60 * 1000,
    enabled: !!userId,
  })
}

export function useCreatePortfolioItem() {
  const queryClient = useQueryClient()
  const uid = useCurrentUserId()

  return useMutation({
    mutationFn: (payload: {
      title: string
      description?: string
      link_url?: string
      position: number
      media?: MediaPayload[]
    }) => createPortfolioItem(payload),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: myPortfolioKey(uid) }),
  })
}

export function useUpdatePortfolioItem() {
  const queryClient = useQueryClient()
  const uid = useCurrentUserId()

  return useMutation({
    mutationFn: ({
      id,
      ...payload
    }: {
      id: string
      title?: string
      description?: string
      link_url?: string
      media?: MediaPayload[]
    }) => updatePortfolioItem(id, payload),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: myPortfolioKey(uid) }),
  })
}

export function useDeletePortfolioItem() {
  const queryClient = useQueryClient()
  const uid = useCurrentUserId()

  return useMutation({
    mutationFn: (id: string) => deletePortfolioItem(id),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: myPortfolioKey(uid) }),
  })
}

export function useReorderPortfolio() {
  const queryClient = useQueryClient()
  const uid = useCurrentUserId()

  return useMutation({
    mutationFn: (itemIds: string[]) => reorderPortfolio(itemIds),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: myPortfolioKey(uid) }),
  })
}

export function useUploadPortfolioImage() {
  return useMutation({
    mutationFn: (file: File) => uploadPortfolioImage(file),
  })
}

export function useUploadPortfolioVideo() {
  return useMutation({
    mutationFn: (file: File) => uploadPortfolioVideo(file),
  })
}
