import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import {
  listAdminMedia,
  getAdminMedia,
  approveMedia,
  rejectMedia,
  deleteMedia,
} from "../api/media-api"
import type { MediaFilters } from "../types"

export function mediaQueryKey(filters: MediaFilters) {
  return ["admin", "media", filters] as const
}

export function useAdminMedia(filters: MediaFilters) {
  return useQuery({
    queryKey: mediaQueryKey(filters),
    queryFn: () => listAdminMedia(filters),
    staleTime: 30 * 1000,
  })
}

export function mediaDetailQueryKey(id: string) {
  return ["admin", "media", id] as const
}

export function useAdminMediaDetail(id: string) {
  return useQuery({
    queryKey: mediaDetailQueryKey(id),
    queryFn: () => getAdminMedia(id),
    enabled: !!id,
  })
}

function useInvalidateMedia(id: string) {
  const queryClient = useQueryClient()
  return () => {
    queryClient.invalidateQueries({ queryKey: ["admin", "media", id] })
    queryClient.invalidateQueries({ queryKey: ["admin", "media"] })
  }
}

export function useApproveMedia(id: string) {
  const invalidate = useInvalidateMedia(id)
  return useMutation({
    mutationFn: () => approveMedia(id),
    onSuccess: invalidate,
  })
}

export function useRejectMedia(id: string) {
  const invalidate = useInvalidateMedia(id)
  return useMutation({
    mutationFn: () => rejectMedia(id),
    onSuccess: invalidate,
  })
}

export function useDeleteMedia(id: string) {
  const invalidate = useInvalidateMedia(id)
  return useMutation({
    mutationFn: () => deleteMedia(id),
    onSuccess: invalidate,
  })
}
