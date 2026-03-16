"use client"

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { useAuth } from "@/shared/hooks/use-auth"
import { getMyProfile, updateProfile } from "../api/profile-api"

const PROFILE_QUERY_KEY = ["profile"]

export function useProfile() {
  const { accessToken } = useAuth()

  return useQuery({
    queryKey: PROFILE_QUERY_KEY,
    queryFn: () => getMyProfile(accessToken!),
    enabled: !!accessToken,
  })
}

export function useUpdateProfile() {
  const { accessToken } = useAuth()
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: Record<string, string>) =>
      updateProfile(accessToken!, data),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: PROFILE_QUERY_KEY }),
  })
}
