"use client"

import { useMutation, useQueryClient } from "@tanstack/react-query"
import {
  uploadPhoto,
  uploadVideo,
  uploadReferrerVideo,
  deleteVideo,
  deleteReferrerVideo,
} from "../api/upload-api"

const PROFILE_QUERY_KEY = ["profile"]

export function useUploadPhoto() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (file: File) => uploadPhoto(file),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: PROFILE_QUERY_KEY }),
  })
}

export function useUploadVideo() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (file: File) => uploadVideo(file),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: PROFILE_QUERY_KEY }),
  })
}

export function useUploadReferrerVideo() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (file: File) => uploadReferrerVideo(file),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: PROFILE_QUERY_KEY }),
  })
}

export function useDeleteVideo() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: () => deleteVideo(),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: PROFILE_QUERY_KEY }),
  })
}

export function useDeleteReferrerVideo() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: () => deleteReferrerVideo(),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: PROFILE_QUERY_KEY }),
  })
}
