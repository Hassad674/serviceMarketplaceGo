"use client"

import { useMutation, useQueryClient } from "@tanstack/react-query"
import {
  uploadPhoto,
  uploadVideo,
  uploadReferrerVideo,
  deleteVideo,
  deleteReferrerVideo,
} from "../api/upload-api"
import { profileQueryKey } from "./use-profile"
import { useCurrentUserId } from "@/shared/hooks/use-current-user-id"

export function useUploadPhoto() {
  const queryClient = useQueryClient()
  const uid = useCurrentUserId()

  return useMutation({
    mutationFn: (file: File) => uploadPhoto(file),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: profileQueryKey(uid) }),
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
