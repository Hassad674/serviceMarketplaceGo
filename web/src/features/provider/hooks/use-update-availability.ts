"use client"

import { useMutation, useQueryClient } from "@tanstack/react-query"
import { useCurrentUserId } from "@/shared/hooks/use-current-user-id"
import {
  updateAvailability,
  type Profile,
  type UpdateAvailabilityInput,
} from "../api/profile-api"
import { profileQueryKey } from "./use-profile"

// Optimistic patch for either the direct or the referrer availability
// slot — never both in the same call. Each page (freelance profile vs
// referral profile) mutates only the field it owns, so fields absent
// from the input are left untouched in both the cache and the backend.
export function useUpdateAvailability() {
  const queryClient = useQueryClient()
  const uid = useCurrentUserId()
  const key = profileQueryKey(uid)

  return useMutation({
    mutationFn: (input: UpdateAvailabilityInput) => updateAvailability(input),
    onMutate: async (input) => {
      await queryClient.cancelQueries({ queryKey: key })
      const previous = queryClient.getQueryData<Profile>(key)
      if (previous) {
        queryClient.setQueryData<Profile>(key, {
          ...previous,
          ...(input.availability_status !== undefined
            ? { availability_status: input.availability_status }
            : {}),
          ...(input.referrer_availability_status !== undefined
            ? {
                referrer_availability_status: input.referrer_availability_status,
              }
            : {}),
        })
      }
      return { previous }
    },
    onError: (_error, _input, context) => {
      if (context?.previous) {
        queryClient.setQueryData<Profile>(key, context.previous)
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: key })
    },
  })
}
