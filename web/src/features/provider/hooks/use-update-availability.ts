"use client"

import { useMutation, useQueryClient } from "@tanstack/react-query"
import { useCurrentUserId } from "@/shared/hooks/use-current-user-id"
import {
  updateAvailability,
  type Profile,
  type UpdateAvailabilityInput,
} from "../api/profile-api"
import { profileQueryKey } from "./use-profile"

// Optimistic mutation for the direct + optional referrer availability.
// The referrer field is only relevant for provider_personal orgs with
// referrer_enabled=true, but we patch it unconditionally — the
// component gates the input, not the cache shape.
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
          availability_status: input.availability_status,
          referrer_availability_status:
            input.referrer_availability_status ?? null,
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
