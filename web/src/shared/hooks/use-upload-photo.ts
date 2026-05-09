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
 *
 * Cache fan-out (avatar-refresh fix, 2026-05-09):
 *   * provider profile cache — what `<UserAvatar>` reads to render
 *     the sidebar + header avatar.
 *   * organization-shared cache — agencies + freelancers also push
 *     the same photo URL into the shared row; refresh it so any
 *     consumer of the new split-profile readers picks up the change.
 *   * profile-completion (every persona variant) — flips the "Photo"
 *     section to filled instantly without waiting for the 30s
 *     staleTime window. Matched on the `["user", uid,
 *     "profile-completion"]` prefix so every cached variant
 *     ("default", "freelance", "referrer") refreshes in one call.
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
      // Organization-shared row stores the same photo URL — refresh
      // it so any reader of the split-profile aggregate stays in sync.
      queryClient.invalidateQueries({
        queryKey: ["user", uid, "organization-shared"],
      })
      // The photo/logo is shared with the client-profile facet
      // (agencies expose both). Invalidate those caches too so an
      // upload done from either page shows up on the other without a
      // manual refresh.
      queryClient.invalidateQueries({ queryKey: ["client-profile"] })
      queryClient.invalidateQueries({ queryKey: ["public-client-profile"] })
      // Profile-completion bar pulls from the same uid-scoped cache
      // family — invalidate by key prefix so every persona variant
      // refreshes (default, freelance, referrer) without a feature
      // import on the profile-completion module.
      queryClient.invalidateQueries({
        queryKey: ["user", uid, "profile-completion"],
      })
    },
  })
}
