"use client"

import { useMutation, useQueryClient } from "@tanstack/react-query"
import {
  uploadVideo,
  uploadReferrerVideo,
  deleteVideo,
  deleteReferrerVideo,
} from "@/shared/lib/upload-api"
import { profileQueryKey } from "./use-profile"
import { useCurrentUserId } from "@/shared/hooks/use-current-user-id"

// `useUploadPhoto` lives in `@/shared/hooks/use-upload-photo` (P9 —
// consumed cross-feature by client-profile). Re-exported here for
// back-compat.
export { useUploadPhoto } from "@/shared/hooks/use-upload-photo"

export function useUploadVideo() {
  const queryClient = useQueryClient()
  const uid = useCurrentUserId()

  return useMutation({
    mutationFn: (file: File) => uploadVideo(file),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: profileQueryKey(uid) }),
  })
}

export function useUploadReferrerVideo() {
  const queryClient = useQueryClient()
  const uid = useCurrentUserId()

  return useMutation({
    mutationFn: (file: File) => uploadReferrerVideo(file),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: profileQueryKey(uid) }),
  })
}

export function useDeleteVideo() {
  const queryClient = useQueryClient()
  const uid = useCurrentUserId()

  return useMutation({
    mutationFn: () => deleteVideo(),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: profileQueryKey(uid) }),
  })
}

export function useDeleteReferrerVideo() {
  const queryClient = useQueryClient()
  const uid = useCurrentUserId()

  return useMutation({
    mutationFn: () => deleteReferrerVideo(),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: profileQueryKey(uid) }),
  })
}
