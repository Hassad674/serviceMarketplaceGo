"use client"

import { useMutation, useQueryClient } from "@tanstack/react-query"
import { uploadPhoto } from "@/shared/lib/upload-api"
import { useCurrentUserId } from "@/shared/hooks/use-current-user-id"

/**
 * Shared profile-photo upload mutation. Lifted out of the `provider`
 * feature (P9) so the `client-profile` feature (and any future
 * profile-facet UI) can wire the upload without importing from
 * provider directly.
 *
 * The mutation invalidates the canonical provider profile query by
 * its key shape, plus the two client-profile caches that mirror the
 * same photo (agencies expose both facets). Hardcoded prefixes are
 * intentional — keeping the shared hook free of feature imports.
 */
export function useUploadPhoto() {
  const queryClient = useQueryClient()
  const uid = useCurrentUserId()

  return useMutation({
    mutationFn: (file: File) => uploadPhoto(file),
    onSuccess: () => {
      // Provider profile cache — the canonical source for /api/v1/profile.
      // Key shape mirrors `profileQueryKey(uid)` in features/provider/hooks/use-profile.
      queryClient.invalidateQueries({ queryKey: ["user", uid, "profile"] })
      // The photo/logo is shared with the client-profile facet
      // (agencies expose both). Invalidate those caches too so an
      // upload done from either page shows up on the other without a
      // manual refresh.
      queryClient.invalidateQueries({ queryKey: ["client-profile"] })
      queryClient.invalidateQueries({ queryKey: ["public-client-profile"] })
    },
  })
}
