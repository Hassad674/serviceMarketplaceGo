"use client"

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { useOrganization } from "@/shared/hooks/use-user"
import {
  fetchPortfolioByOrganization,
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

function myPortfolioKey(orgId: string | undefined) {
  return ["portfolio", "org", orgId, "mine"] as const
}

export function useMyPortfolio() {
  const { data: org } = useOrganization()
  const orgId = org?.id

  return useQuery({
    queryKey: myPortfolioKey(orgId),
    queryFn: async () => {
      if (!orgId) return { data: [], next_cursor: "", has_more: false }
      return fetchPortfolioByOrganization(orgId)
    },
    staleTime: 2 * 60 * 1000,
    enabled: !!orgId,
  })
}

export function usePortfolioByOrganization(orgId: string) {
  return useQuery({
    queryKey: ["portfolio", "org", orgId],
    queryFn: () => fetchPortfolioByOrganization(orgId),
    staleTime: 2 * 60 * 1000,
    enabled: !!orgId,
  })
}

export function useCreatePortfolioItem() {
  const queryClient = useQueryClient()
  const { data: org } = useOrganization()
  const orgId = org?.id

  return useMutation({
    mutationFn: (payload: {
      title: string
      description?: string
      link_url?: string
      position: number
      media?: MediaPayload[]
    }) => createPortfolioItem(payload),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: myPortfolioKey(orgId) }),
  })
}

export function useUpdatePortfolioItem() {
  const queryClient = useQueryClient()
  const { data: org } = useOrganization()
  const orgId = org?.id

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
      queryClient.invalidateQueries({ queryKey: myPortfolioKey(orgId) }),
  })
}

export function useDeletePortfolioItem() {
  const queryClient = useQueryClient()
  const { data: org } = useOrganization()
  const orgId = org?.id

  return useMutation({
    mutationFn: (id: string) => deletePortfolioItem(id),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: myPortfolioKey(orgId) }),
  })
}

export function useReorderPortfolio() {
  const queryClient = useQueryClient()
  const { data: org } = useOrganization()
  const orgId = org?.id

  return useMutation({
    mutationFn: (itemIds: string[]) => reorderPortfolio(itemIds),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: myPortfolioKey(orgId) }),
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
