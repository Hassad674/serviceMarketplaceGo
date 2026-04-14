"use client"

import { useMutation, useQueryClient } from "@tanstack/react-query"
import { useCurrentUserId } from "@/shared/hooks/use-current-user-id"
import {
  updateOrganizationPhoto,
  type OrganizationSharedProfile,
} from "../api/organization-shared-api"
import { uploadOrganizationPhoto } from "../api/photo-upload-api"
import { organizationSharedQueryKey } from "./use-organization-shared"
import { invalidateSharedDependents } from "./use-update-organization-location"

// useUploadOrganizationPhoto orchestrates the two-step photo flow:
//   1. multipart upload → backend returns the canonical URL
//   2. PUT /organization/photo → stamps the URL onto the org row
//
// Keeping both steps inside one mutation means the parent UI only
// wires a single loading/error surface and the cache fan-out happens
// in exactly one place.
export function useUploadOrganizationPhoto() {
  const queryClient = useQueryClient()
  const uid = useCurrentUserId()
  const sharedKey = organizationSharedQueryKey(uid)

  return useMutation({
    mutationFn: async (file: File) => {
      const { url } = await uploadOrganizationPhoto(file)
      return updateOrganizationPhoto({ photo_url: url })
    },
    onSuccess: (next) => {
      queryClient.setQueryData<OrganizationSharedProfile>(sharedKey, next)
      invalidateSharedDependents(queryClient, uid)
    },
  })
}
