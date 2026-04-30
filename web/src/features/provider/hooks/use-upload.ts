"use client"

import { useMutation, useQueryClient } from "@tanstack/react-query"
import {
  uploadPhoto,
  uploadVideo,
  uploadReferrerVideo,
  deleteVideo,
  deleteReferrerVideo,
} from "@/shared/lib/upload-api"
import { profileQueryKey } from "./use-profile"
import { useCurrentUserId } from "@/shared/hooks/use-current-user-id"

export function useUploadPhoto() {
  const queryClient = useQueryClient()
  const uid = useCurrentUserId()

  return useMutation({
    mutationFn: (file: File) => uploadPhoto(file),
    onSuccess: () => {
      // Provider profile cache — the canonical source for /api/v1/profile.
      queryClient.invalidateQueries({ queryKey: profileQueryKey(uid) })
      // The photo/logo is shared with the client-profile facet
      // (agencies expose both). Invalidate the client-profile cache
      // too so an upload done from either page shows up on the other
      // without a manual refresh. We keep the provider feature free
      // of direct client-profile imports by keying on the query key
      // prefix rather than pulling the hook's constant.
      queryClient.invalidateQueries({ queryKey: ["client-profile"] })
      queryClient.invalidateQueries({ queryKey: ["public-client-profile"] })
    },
  })
}

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
