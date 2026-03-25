"use client"

import { useMutation, useQueryClient } from "@tanstack/react-query"
import { useAuth } from "@/shared/hooks/use-auth"
import {
  uploadPhoto,
  uploadVideo,
  uploadReferrerVideo,
} from "../api/upload-api"

const PROFILE_QUERY_KEY = ["profile"]

export function useUploadPhoto() {
  const { accessToken } = useAuth()
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (file: File) => uploadPhoto(accessToken!, file),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: PROFILE_QUERY_KEY }),
  })
}

export function useUploadVideo() {
  const { accessToken } = useAuth()
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (file: File) => uploadVideo(accessToken!, file),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: PROFILE_QUERY_KEY }),
  })
}

export function useUploadReferrerVideo() {
  const { accessToken } = useAuth()
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (file: File) => uploadReferrerVideo(accessToken!, file),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: PROFILE_QUERY_KEY }),
  })
}
