"use client"

import { useMutation, useQueryClient } from "@tanstack/react-query"
import { useCurrentUserId } from "@/shared/hooks/use-current-user-id"
import {
  deleteFreelanceVideo,
  uploadFreelanceVideo,
} from "../api/freelance-video-api"
import { freelanceProfileQueryKey } from "./use-freelance-profile"

// Upload and delete mutations for the freelance presentation video.
// Both invalidate the freelance profile cache so the embedded
// video_url reflects the post-mutation state. We also invalidate the
// organization-shared cache because the legacy upload handler still
// stamps the URL on the legacy profiles row — the next refetch will
// pick up whatever the backend reconciled.
export function useUploadFreelanceVideo() {
  const queryClient = useQueryClient()
  const uid = useCurrentUserId()
  return useMutation({
    mutationFn: (file: File) => uploadFreelanceVideo(file),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: freelanceProfileQueryKey(uid) })
    },
  })
}

export function useDeleteFreelanceVideo() {
  const queryClient = useQueryClient()
  const uid = useCurrentUserId()
  return useMutation({
    mutationFn: () => deleteFreelanceVideo(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: freelanceProfileQueryKey(uid) })
    },
  })
}
