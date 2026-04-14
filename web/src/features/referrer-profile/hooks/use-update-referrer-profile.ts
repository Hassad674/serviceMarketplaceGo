"use client"

import { useMutation, useQueryClient } from "@tanstack/react-query"
import { useCurrentUserId } from "@/shared/hooks/use-current-user-id"
import {
  updateReferrerAvailability,
  updateReferrerExpertise,
  updateReferrerProfile,
  type AvailabilityStatus,
  type ReferrerProfile,
  type UpdateReferrerProfileInput,
} from "../api/referrer-profile-api"
import { referrerProfileQueryKey } from "./use-referrer-profile"

export function useUpdateReferrerProfile() {
  const queryClient = useQueryClient()
  const uid = useCurrentUserId()
  const key = referrerProfileQueryKey(uid)

  return useMutation({
    mutationFn: (input: UpdateReferrerProfileInput) =>
      updateReferrerProfile(input),
    onSuccess: (next) => {
      queryClient.setQueryData<ReferrerProfile>(key, next)
    },
  })
}

export function useUpdateReferrerAvailability() {
  const queryClient = useQueryClient()
  const uid = useCurrentUserId()
  const key = referrerProfileQueryKey(uid)

  return useMutation({
    mutationFn: (status: AvailabilityStatus) =>
      updateReferrerAvailability(status),
    onSuccess: (next) => {
      queryClient.setQueryData<ReferrerProfile>(key, next)
    },
  })
}

export function useUpdateReferrerExpertise() {
  const queryClient = useQueryClient()
  const uid = useCurrentUserId()
  const key = referrerProfileQueryKey(uid)

  return useMutation({
    mutationFn: (domains: string[]) => updateReferrerExpertise(domains),
    onSuccess: (next) => {
      queryClient.setQueryData<ReferrerProfile>(key, next)
    },
  })
}
