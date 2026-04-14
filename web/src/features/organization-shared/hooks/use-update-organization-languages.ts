"use client"

import { useMutation, useQueryClient } from "@tanstack/react-query"
import { useCurrentUserId } from "@/shared/hooks/use-current-user-id"
import {
  updateOrganizationLanguages,
  type OrganizationSharedProfile,
  type UpdateOrganizationLanguagesInput,
} from "../api/organization-shared-api"
import { organizationSharedQueryKey } from "./use-organization-shared"
import { invalidateSharedDependents } from "./use-update-organization-location"

export function useUpdateOrganizationLanguages() {
  const queryClient = useQueryClient()
  const uid = useCurrentUserId()
  const sharedKey = organizationSharedQueryKey(uid)

  return useMutation({
    mutationFn: (input: UpdateOrganizationLanguagesInput) =>
      updateOrganizationLanguages(input),
    onSuccess: (next) => {
      queryClient.setQueryData<OrganizationSharedProfile>(sharedKey, next)
      invalidateSharedDependents(queryClient, uid)
    },
  })
}
