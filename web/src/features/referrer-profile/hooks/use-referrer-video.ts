"use client"

import { useMutation, useQueryClient } from "@tanstack/react-query"
import { useCurrentUserId } from "@/shared/hooks/use-current-user-id"
import {
  deleteReferrerVideo,
  uploadReferrerVideo,
} from "../api/referrer-video-api"
import { referrerProfileQueryKey } from "./use-referrer-profile"

export function useUploadReferrerVideo() {
  const queryClient = useQueryClient()
  const uid = useCurrentUserId()
  return useMutation({
    mutationFn: (file: File) => uploadReferrerVideo(file),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: referrerProfileQueryKey(uid) })
    },
  })
}

export function useDeleteReferrerVideo() {
  const queryClient = useQueryClient()
  const uid = useCurrentUserId()
  return useMutation({
    mutationFn: () => deleteReferrerVideo(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: referrerProfileQueryKey(uid) })
    },
  })
}
