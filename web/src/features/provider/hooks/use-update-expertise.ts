"use client"

import { useMutation, useQueryClient } from "@tanstack/react-query"
import { updateExpertiseDomains } from "../api/expertise-api"
import { profileQueryKey } from "./use-profile"
import { useCurrentUserId } from "@/shared/hooks/use-current-user-id"
import type { Profile } from "../api/profile-api"

// Optimistic mutation for the current operator's expertise list.
//
// Flow:
//   1. Snapshot the current profile from the cache.
//   2. Patch the cached profile with the new ordered domain list so the
//      UI flips to the new state immediately.
//   3. Send the PUT. On success we trust the server's canonical order
//      and write it back (may differ from client order if the backend
//      normalized anything). On error we restore the snapshot so the
//      editor reverts and can surface the failure.
//
// The server response is the source of truth — the optimistic patch is
// only a latency hiding trick. That's why `onSuccess` writes the
// returned array instead of leaving the optimistic patch in place.
export function useUpdateExpertiseDomains() {
  const queryClient = useQueryClient()
  const uid = useCurrentUserId()
  const key = profileQueryKey(uid)

  return useMutation({
    mutationFn: (domains: string[]) => updateExpertiseDomains(domains),
    onMutate: async (domains) => {
      await queryClient.cancelQueries({ queryKey: key })
      const previous = queryClient.getQueryData<Profile>(key)
      if (previous) {
        queryClient.setQueryData<Profile>(key, {
          ...previous,
          expertise_domains: domains,
        })
      }
      return { previous }
    },
    onError: (_error, _domains, context) => {
      if (context?.previous) {
        queryClient.setQueryData<Profile>(key, context.previous)
      }
    },
    onSuccess: (result) => {
      const current = queryClient.getQueryData<Profile>(key)
      if (current) {
        queryClient.setQueryData<Profile>(key, {
          ...current,
          expertise_domains: result.expertise_domains,
        })
      }
    },
  })
}
