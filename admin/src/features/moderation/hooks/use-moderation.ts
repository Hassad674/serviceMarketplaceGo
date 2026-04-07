import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import {
  listModerationItems,
  approveMedia,
  rejectMedia,
  deleteMedia,
  approveMessageModeration,
  hideMessage,
  approveReviewModeration,
  deleteReview,
  resolveReport,
} from "../api/moderation-api"
import type { ModerationFilters } from "../types"

export function moderationQueryKey(filters: ModerationFilters) {
  return ["admin", "moderation", filters] as const
}

export function useModerationItems(filters: ModerationFilters) {
  return useQuery({
    queryKey: moderationQueryKey(filters),
    queryFn: () => listModerationItems(filters),
    staleTime: 30 * 1000,
  })
}

function useInvalidateModeration() {
  const queryClient = useQueryClient()
  return () => {
    queryClient.invalidateQueries({ queryKey: ["admin", "moderation"] })
    queryClient.invalidateQueries({ queryKey: ["admin", "moderation-count"] })
  }
}

export function useApproveMedia() {
  const invalidate = useInvalidateModeration()
  return useMutation({
    mutationFn: (id: string) => approveMedia(id),
    onSuccess: invalidate,
  })
}

export function useRejectMedia() {
  const invalidate = useInvalidateModeration()
  return useMutation({
    mutationFn: (id: string) => rejectMedia(id),
    onSuccess: invalidate,
  })
}

export function useDeleteMedia() {
  const invalidate = useInvalidateModeration()
  return useMutation({
    mutationFn: (id: string) => deleteMedia(id),
    onSuccess: invalidate,
  })
}

export function useApproveMessageModeration() {
  const invalidate = useInvalidateModeration()
  return useMutation({
    mutationFn: (id: string) => approveMessageModeration(id),
    onSuccess: invalidate,
  })
}

export function useHideMessage() {
  const invalidate = useInvalidateModeration()
  return useMutation({
    mutationFn: (id: string) => hideMessage(id),
    onSuccess: invalidate,
  })
}

export function useApproveReviewModeration() {
  const invalidate = useInvalidateModeration()
  return useMutation({
    mutationFn: (id: string) => approveReviewModeration(id),
    onSuccess: invalidate,
  })
}

export function useDeleteReview() {
  const invalidate = useInvalidateModeration()
  return useMutation({
    mutationFn: (id: string) => deleteReview(id),
    onSuccess: invalidate,
  })
}

export function useResolveReport() {
  const invalidate = useInvalidateModeration()
  return useMutation({
    mutationFn: (params: { reportId: string; status: "resolved" | "dismissed"; adminNote: string }) =>
      resolveReport(params.reportId, { status: params.status, admin_note: params.adminNote }),
    onSuccess: invalidate,
  })
}
