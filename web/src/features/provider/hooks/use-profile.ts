"use client"

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { getMyProfile, updateProfile } from "../api/profile-api"

const PROFILE_QUERY_KEY = ["profile"]

export function useProfile() {
  return useQuery({
    queryKey: PROFILE_QUERY_KEY,
    queryFn: () => getMyProfile(),
  })
}

export function useUpdateProfile() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: Record<string, string>) =>
      updateProfile(data),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: PROFILE_QUERY_KEY }),
  })
}
