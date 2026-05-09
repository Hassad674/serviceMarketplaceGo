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
//
// Cache fan-out (avatar-refresh fix, 2026-05-09):
//   * organization-shared cache → setQueryData with the fresh row
//   * shared dependents (freelance-profile, referrer-profile) →
//     invalidate so the page hero re-fetches its joined view
//   * legacy provider profile cache (`["user", uid, "profile"]`) →
//     invalidate because <UserAvatar> reads `photo_url` from there;
//     without this the sidebar identity card and the header
//     dropdown still render the OLD photo (or a Portrait fallback)
//     after a successful upload on /profile or /referral.
//   * client-profile facets → share the photo with the provider side.
//   * profile-completion → invalidate every persona variant so the
//     "Photo" section flips to filled instantly without waiting for
//     the 30-second staleTime window.
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
      queryClient.invalidateQueries({ queryKey: ["user", uid, "profile"] })
      queryClient.invalidateQueries({ queryKey: ["client-profile"] })
      queryClient.invalidateQueries({ queryKey: ["public-client-profile"] })
      queryClient.invalidateQueries({
        queryKey: ["user", uid, "profile-completion"],
      })
    },
  })
}
