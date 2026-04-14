"use client"

import { useMutation, useQueryClient } from "@tanstack/react-query"
import { useCurrentUserId } from "@/shared/hooks/use-current-user-id"
import {
  updateLocation,
  type Profile,
  type UpdateLocationInput,
} from "../api/profile-api"
import { profileQueryKey } from "./use-profile"

// Optimistic mutation for the org's location block (city, country,
// work modes, travel radius). Same pattern as useUpdateExpertiseDomains
// — patch the cache synchronously so the UI reflects the new values
// immediately, roll back on error, invalidate on success so we pick up
// server-side geocoding (latitude/longitude) the next render.
export function useUpdateLocation() {
  const queryClient = useQueryClient()
  const uid = useCurrentUserId()
  const key = profileQueryKey(uid)

  return useMutation({
    mutationFn: (input: UpdateLocationInput) => updateLocation(input),
    onMutate: async (input) => {
      await queryClient.cancelQueries({ queryKey: key })
      const previous = queryClient.getQueryData<Profile>(key)
      if (previous) {
        queryClient.setQueryData<Profile>(key, {
          ...previous,
          city: input.city,
          country_code: input.country_code,
          work_mode: input.work_mode,
          travel_radius_km: input.travel_radius_km,
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
      // Refetch so we also pick up the server-side geocoded coordinates.
      queryClient.invalidateQueries({ queryKey: key })
    },
  })
}
