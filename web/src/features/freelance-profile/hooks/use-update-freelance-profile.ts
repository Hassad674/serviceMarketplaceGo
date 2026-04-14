"use client"

import { useMutation, useQueryClient } from "@tanstack/react-query"
import { useCurrentUserId } from "@/shared/hooks/use-current-user-id"
import {
  updateFreelanceAvailability,
  updateFreelanceExpertise,
  updateFreelanceProfile,
  type AvailabilityStatus,
  type FreelanceProfile,
  type UpdateFreelanceProfileInput,
} from "../api/freelance-profile-api"
import { freelanceProfileQueryKey } from "./use-freelance-profile"

// Core profile update: title / about / video_url. The backend returns
// the refreshed aggregate so we can cache-seed without a second
// network round-trip.
export function useUpdateFreelanceProfile() {
  const queryClient = useQueryClient()
  const uid = useCurrentUserId()
  const key = freelanceProfileQueryKey(uid)

  return useMutation({
    mutationFn: (input: UpdateFreelanceProfileInput) =>
      updateFreelanceProfile(input),
    onSuccess: (next) => {
      queryClient.setQueryData<FreelanceProfile>(key, next)
    },
  })
}

// Availability is a single-field write; kept as its own hook so the
// availability UI component can surface its own loading state without
// touching the broader profile form.
export function useUpdateFreelanceAvailability() {
  const queryClient = useQueryClient()
  const uid = useCurrentUserId()
  const key = freelanceProfileQueryKey(uid)

  return useMutation({
    mutationFn: (status: AvailabilityStatus) =>
      updateFreelanceAvailability(status),
    onSuccess: (next) => {
      queryClient.setQueryData<FreelanceProfile>(key, next)
    },
  })
}

// Expertise replaces the domain list atomically.
export function useUpdateFreelanceExpertise() {
  const queryClient = useQueryClient()
  const uid = useCurrentUserId()
  const key = freelanceProfileQueryKey(uid)

  return useMutation({
    mutationFn: (domains: string[]) => updateFreelanceExpertise(domains),
    onSuccess: (next) => {
      queryClient.setQueryData<FreelanceProfile>(key, next)
    },
  })
}
