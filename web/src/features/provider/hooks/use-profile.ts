"use client"

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { getMyProfile, updateProfile } from "../api/profile-api"
import { useCurrentUserId } from "@/shared/hooks/use-current-user-id"

export function profileQueryKey(uid: string | undefined) {
  return ["user", uid, "profile"] as const
}

export function useProfile() {
  const uid = useCurrentUserId()

  return useQuery({
    queryKey: profileQueryKey(uid),
    queryFn: () => getMyProfile(),
    staleTime: 5 * 60 * 1000, // 5 minutes — own profile data rarely changes externally
  })
}

export function useUpdateProfile() {
  const queryClient = useQueryClient()
  const uid = useCurrentUserId()

  return useMutation({
    mutationFn: (data: Record<string, string>) =>
      updateProfile(data),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: profileQueryKey(uid) }),
  })
}
